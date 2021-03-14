package manager

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/combust-labs/firebox/api/models"
	"github.com/combust-labs/firebox/config"
	"github.com/combust-labs/firebox/pkg/actors/vmm"
	"github.com/combust-labs/firebox/pkg/log"
	"github.com/pkg/errors"
	"io"
	"io/ioutil"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

type VMMManager struct {
	actor.Actor

	init sync.Once

	logger    *log.Logger
	vmmConfig config.VMMConfig
	db        *db

	rootContext *actor.RootContext
	self        *actor.PID
}

func NewVMMManager(logger *log.Logger, vmmConfig config.VMMConfig) *VMMManager {
	return &VMMManager{
		logger:    logger,
		vmmConfig: vmmConfig,
		db:        initdb(),
	}
}

func (m *VMMManager) Init(system *actor.ActorSystem) {
	m.init.Do(func() {
		props := actor.PropsFromProducer(func() actor.Actor { return m })
		m.rootContext = system.Root
		m.self = system.Root.SpawnPrefix(props, "vmm-manager")
	})
}

func (m *VMMManager) Receive(context actor.Context) {
	switch msg := context.Message().(type) {
	case *vmm.Stopped:
		m.logger.Infof("Unexpected termination vmid %v", msg.ID)
		m.db.del(msg.ID)
	case *vmm.Ready:
		m.logger.Infof("Machine READY vmid: %v, ip: %v", msg.ID, msg.IP)
		m.db.ready(msg.ID, true)
	case *vmm.Unready:
		m.logger.Warn("Machine UNREADY vmid: %v, ip: %v", msg.ID, msg.IP)
		m.db.ready(msg.ID, false)
	}
}

func (m *VMMManager) StartVMM() (*vmm.Metadata, error) {
	props := actor.PropsFromProducer(func() actor.Actor { return vmm.NewVMMActor(m.logger, m.vmmConfig) })
	pid := m.rootContext.SpawnPrefix(props, "vmm/")

	timeout := 30 * time.Second
	// timeout can leave orphan (not referenced) VMM
	startResult, err := m.rootContext.RequestFuture(pid, &vmm.Start{Manager: m.self}, timeout).Result()
	if err != nil {
		return nil, err
	}

	switch msg := startResult.(type) {
	case *vmm.Started:
		if err := m.db.add(msg.ID, pid, msg.IP); err != nil {
			// should never happen, otherwise the vmm should be stopped
			return nil, err
		}
		return &msg.Metadata, nil

	case *vmm.Failure:
		return nil, msg.Err
	default:
		return nil, errors.Errorf("Internal error: unexpected message: %v", msg)
	}
}

func (m *VMMManager) Close() error {
	m.logger.Info("Stopping all VMMs")

	timeout := 1 * time.Second
	entries := m.db.entries()
	m.logger.Infof("Machines to stop %v", len(entries))
	for _, entry := range entries {
		m.logger.Infof("Sending stop pid %s vmid %s", entry.pid, entry.vmid)
		_, err := m.rootContext.RequestFuture(entry.pid, &vmm.Stop{}, timeout).Result()
		if err != nil {
			m.logger.Infof("Failed to stop vmid %v: %v", entry.vmid, err)
		}
		m.db.del(entry.vmid)
	}
	return nil
}

func (m *VMMManager) InvokeHTTP(request *models.HTTPRequest) (*models.HTTPResponse, error) {
	ip, err := m.getServiceIP()
	if err != nil {
		return nil, err
	}
	return m.invokeService(ip, request)
}

func (m *VMMManager) getServiceIP() (net.IP, error) {
	// naive random LB

	ready := make([]entry, 0)
	for _, r := range m.db.entries() {
		if r.ready {
			ready = append(ready, r)
		}
	}
	l := len(ready)

	if l == 0 {
		return nil, errors.New("No READY machine found")
	}
	r := rand.Intn(l)
	return ready[r].ip, nil
}

func (m *VMMManager) invokeService(ip net.IP, req *models.HTTPRequest) (*models.HTTPResponse, error) {
	httpRequest, err := toHttpRequest(context.Background(), "http", ip.String(), 8080, req)
	if err != nil {
		return nil, err
	}
	m.logger.Infof("Sending HTTP request to %s: %s %s", httpRequest.Host, httpRequest.Method, httpRequest.URL)

	resp, err := http.DefaultClient.Do(httpRequest)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return toHttpResponse(resp)
}

func toHttpResponse(resp *http.Response) (*models.HTTPResponse, error) {

	payload, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	const isBase64Encoded = true
	body := base64.StdEncoding.EncodeToString(payload)
	cookies := make([]string, 0)
	for _, cookie := range resp.Cookies() {
		cookies = append(cookies, cookie.String())
	}

	var headers map[string]string
	var multiValueHeaders map[string][]string

	for k, vs := range resp.Header {
		if len(vs) == 1 {
			if headers == nil {
				headers = make(map[string]string)
			}
			headers[k] = vs[0]
		} else {
			if multiValueHeaders == nil {
				multiValueHeaders = make(map[string][]string)
			}
			multiValueHeaders[k] = vs
		}
	}

	result := &models.HTTPResponse{
		StatusCode:        int64(resp.StatusCode),
		Body:              body,
		Cookies:           cookies,
		Headers:           headers,
		IsBase64Encoded:   isBase64Encoded,
		MultiValueHeaders: multiValueHeaders,
	}

	return result, nil
}

func toHttpRequest(ctx context.Context, schema string, ip string, port int, req *models.HTTPRequest) (*http.Request, error) {
	var (
		bodyReader io.Reader
	)
	if req.Body != "" && req.IsBase64Encoded {
		b, err := base64.StdEncoding.DecodeString(req.Body)
		if err != nil {
			return nil, err
		}
		bodyReader = bytes.NewReader(b)
	} else {
		bodyReader = strings.NewReader(req.Body)
	}
	u := url.URL{
		Scheme:   schema,
		Host:     fmt.Sprintf("%s:%d", ip, port),
		RawQuery: req.RawQueryString,
		Path:     req.RawPath,
	}
	request, err := http.NewRequestWithContext(ctx, req.HTTPMethod, u.String(), bodyReader)
	if err != nil {
		return nil, err
	}
	for _, cookie := range req.Cookies {
		request.Header.Add("Cookie", cookie)
	}
	for k, v := range req.Headers {
		request.Header.Add(k, v)
	}
	for k, vs := range req.MultiValueHeaders {
		for _, v := range vs {
			request.Header.Add(k, v)
		}
	}
	return request, nil
}
