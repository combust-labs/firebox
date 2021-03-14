package handlers

import (
	"github.com/combust-labs/firebox/api/models"
	"github.com/combust-labs/firebox/api/server/restapi/service"
	"github.com/combust-labs/firebox/pkg/actors/manager"
	"github.com/combust-labs/firebox/pkg/log"
	"github.com/go-openapi/runtime/middleware"
	"github.com/pkg/errors"
)

func NewServiceInvokeHandler(logger *log.Logger, manager *manager.VMMManager) service.InvokeHandler {
	return &serviceInvokeHandler{
		logger:  logger,
		manager: manager,
	}
}

type serviceInvokeHandler struct {
	logger  *log.Logger
	manager *manager.VMMManager
}

func (h *serviceInvokeHandler) Handle(params service.InvokeParams) middleware.Responder {
	resp, err := h.manager.InvokeHTTP(params.Data)
	if err != nil {
		// it could be also 500
		return service.NewInvokeServiceUnavailable().WithPayload(&models.StandardError{
			Code:    503,
			Message: errors.Wrap(err, "HTTP invocation error").Error(),
		})
	}
	return service.NewInvokeOK().WithPayload(resp)
}
