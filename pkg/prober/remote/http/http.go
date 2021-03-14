package http

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/pkg/errors"
)

const (
	userAgent    = "firebox"
	majorVersion = "1"
	minorVersion = "0"
)

type Prober interface {
	Probe(url *url.URL, headers http.Header, timeout time.Duration) error
}

func New() Prober {
	tlsConfig := &tls.Config{InsecureSkipVerify: true}
	return NewWithTLSConfig(tlsConfig)
}

func NewWithTLSConfig(config *tls.Config) Prober {
	// see some defaults from http.DefaultTransport.(*http.Transport)
	transport := &http.Transport{
		Proxy:             http.ProxyURL(nil),
		TLSClientConfig:   config,
		DisableKeepAlives: true,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
	return httpProber{transport}
}

type httpProber struct {
	transport *http.Transport
}

func (pr httpProber) Probe(url *url.URL, headers http.Header, timeout time.Duration) error {
	client := &http.Client{
		Timeout:   timeout,
		Transport: pr.transport,
	}
	return DoHTTPProbe(url, headers, client)
}

type GetHTTPInterface interface {
	Do(req *http.Request) (*http.Response, error)
}

func DoHTTPProbe(url *url.URL, headers http.Header, client GetHTTPInterface) error {
	req, err := http.NewRequest("GET", url.String(), nil)
	if err != nil {
		return err
	}
	if _, ok := headers["User-Agent"]; !ok {
		if headers == nil {
			headers = http.Header{}
		}
		headers.Set("User-Agent", fmt.Sprintf("%s/%s.%s", userAgent, majorVersion, minorVersion))
	}
	req.Header = headers
	if headers.Get("Host") != "" {
		req.Host = headers.Get("Host")
	}
	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode >= http.StatusOK && res.StatusCode < http.StatusMultipleChoices {
		return nil
	}
	return errors.Errorf("HTTP probe failed with statuscode: %d", res.StatusCode)
}
