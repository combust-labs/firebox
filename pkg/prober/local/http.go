package local

import (
	"github.com/combust-labs/firebox/api/server/restapi/health"
	"github.com/go-openapi/runtime/middleware"
	"go.uber.org/atomic"
)

type HTTPProbe struct {
	ready   *httpReadyHandler
	healthy *httpHealthyHandler
}

func NewHTTP() *HTTPProbe {
	return &HTTPProbe{
		ready:   &httpReadyHandler{},
		healthy: &httpHealthyHandler{},
	}
}

func (p *HTTPProbe) HealthyHandler() health.IsHealthyHandler {
	return p.healthy
}

func (p *HTTPProbe) ReadyHandler() health.IsReadyHandler {
	return p.ready
}

type httpHealthyHandler struct {
	ok atomic.Bool
}

func (f httpHealthyHandler) Handle(_ health.IsHealthyParams) middleware.Responder {
	if f.ok.Load() {
		return health.NewIsHealthyOK()
	} else {
		return health.NewIsHealthyServiceUnavailable()
	}
}

type httpReadyHandler struct {
	ok atomic.Bool
}

func (f httpReadyHandler) Handle(_ health.IsReadyParams) middleware.Responder {
	if f.ok.Load() {
		return health.NewIsReadyOK()
	} else {
		return health.NewIsReadyServiceUnavailable()
	}
}

func (p HTTPProbe) IsReady() bool {
	return p.ready.ok.Load()
}

func (p *HTTPProbe) SetReady() {
	p.ready.ok.Store(true)
}

func (p *HTTPProbe) SetNotReady(_ error) {
	p.ready.ok.Store(false)
}

func (p *HTTPProbe) IsHealthy() bool {
	return p.healthy.ok.Load()
}

func (p *HTTPProbe) SetHealthy() {
	p.healthy.ok.Store(true)
}

func (p *HTTPProbe) SetNotHealthy(_ error) {
	p.healthy.ok.Store(false)
}
