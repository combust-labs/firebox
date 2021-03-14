// Code generated by go-swagger; DO NOT EDIT.

package vm

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the generate command

import (
	"net/http"

	"github.com/go-openapi/runtime/middleware"
)

// PostVMRunHandlerFunc turns a function with the right signature into a post VM run handler
type PostVMRunHandlerFunc func(PostVMRunParams) middleware.Responder

// Handle executing the request and returning a response
func (fn PostVMRunHandlerFunc) Handle(params PostVMRunParams) middleware.Responder {
	return fn(params)
}

// PostVMRunHandler interface for that can handle valid post VM run params
type PostVMRunHandler interface {
	Handle(PostVMRunParams) middleware.Responder
}

// NewPostVMRun creates a new http.Handler for the post VM run operation
func NewPostVMRun(ctx *middleware.Context, handler PostVMRunHandler) *PostVMRun {
	return &PostVMRun{Context: ctx, Handler: handler}
}

/* PostVMRun swagger:route POST /vm/run vm postVmRun

This endpoint creates a new VM and starts it

*/
type PostVMRun struct {
	Context *middleware.Context
	Handler PostVMRunHandler
}

func (o *PostVMRun) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	route, rCtx, _ := o.Context.RouteInfo(r)
	if rCtx != nil {
		r = rCtx
	}
	var Params = NewPostVMRunParams()
	if err := o.Context.BindValidRequest(r, route, &Params); err != nil { // bind params
		o.Context.Respond(rw, r, route.Produces, route, err)
		return
	}

	res := o.Handler.Handle(Params) // actually handle the request
	o.Context.Respond(rw, r, route.Produces, route, res)

}