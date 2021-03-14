package vmm

import (
	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/combust-labs/firebox/pkg/actors/ticker"
	"github.com/combust-labs/firebox/pkg/log"
	httpprober "github.com/combust-labs/firebox/pkg/prober/remote/http"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

type HTTPGetAction struct {
	Path   string
	Port   int
	Host   string
	Scheme string
}

type ProbeSpec struct {
	HTTPGet             HTTPGetAction
	InitialDelaySeconds int32
	TimeoutSeconds      int32
	PeriodSeconds       int32
	SuccessThreshold    int32
	FailureThreshold    int32
}

func TickerFunc(logger *log.Logger, spec ProbeSpec, context actor.Context, readinessPID *actor.PID, metadata Metadata) ticker.HandlerFunc {
	prober := httpprober.New()
	probeUrl := formatURL(spec.HTTPGet.Scheme, spec.HTTPGet.Host, spec.HTTPGet.Port, spec.HTTPGet.Path)
	headers := make(http.Header)
	return func() {
		logger.Debugf("Probe to %s", probeUrl)
		err := prober.Probe(probeUrl, headers, time.Duration(spec.TimeoutSeconds)*time.Second)
		logger.Debugf("Probe result err %v", err)
		if err != nil {
			context.Send(readinessPID, &Unready{Metadata: metadata})
		} else {
			context.Send(readinessPID, &Ready{Metadata: metadata})
		}
	}
}

func formatURL(scheme string, host string, port int, path string) *url.URL {
	u, err := url.Parse(path)
	if err != nil {
		u = &url.URL{
			Path: path,
		}
	}
	u.Scheme = scheme
	u.Host = net.JoinHostPort(host, strconv.Itoa(port))
	return u
}
