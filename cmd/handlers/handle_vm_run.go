package handlers

import (
	"github.com/combust-labs/firebox/api/models"
	"github.com/combust-labs/firebox/api/server/restapi/vm"
	"github.com/combust-labs/firebox/pkg/actors/manager"
	"github.com/combust-labs/firebox/pkg/log"
	"github.com/go-openapi/runtime/middleware"
	"github.com/pkg/errors"
)

func NewVMPostVMRunHandler(logger *log.Logger, manager *manager.VMMManager) *VMPostVMRunHandler {
	return &VMPostVMRunHandler{
		logger:  logger,
		manager: manager,
	}
}

type VMPostVMRunHandler struct {
	logger  *log.Logger
	manager *manager.VMMManager
}

func (h *VMPostVMRunHandler) Handle(_ vm.PostVMRunParams) middleware.Responder {
	machine, err := h.manager.StartVMM()
	if err != nil {
		err = errors.Wrap(err, "StartVMM failed")
		h.logger.Errorf("%v", err)
		return vm.NewPostVMRunInternalServerError().WithPayload(&models.StandardError{
			Code:    500,
			Message: err.Error(),
		})
	}
	return vm.NewPostVMRunOK().WithPayload(&models.VM{
		ID: machine.ID,
		IP: machine.IP.String(),
	})
}
