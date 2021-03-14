// This file is safe to edit. Once it exists it will not be overwritten

package server

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"

	"github.com/go-openapi/errors"
	"github.com/go-openapi/runtime"
	"github.com/go-openapi/runtime/middleware"

	"github.com/combust-labs/firebox/api/server/restapi"
	"github.com/combust-labs/firebox/api/server/restapi/health"
	"github.com/combust-labs/firebox/api/server/restapi/service"
	"github.com/combust-labs/firebox/api/server/restapi/vm"
	"github.com/combust-labs/firebox/pkg/prober"
	"github.com/combust-labs/firebox/pkg/prober/local"
)

//go:generate swagger generate server --target ../../api --name Firebox --spec ../swagger.yaml --api-package restapi --server-package server --principal interface{} --exclude-main

var (
	// ServerCtx and ServerCancel
	ServerCtx, serverCancel = context.WithCancel(context.Background())

	HTTPProber   = local.NewHTTP()
	StatusProber = prober.Combine(HTTPProber)
)

func configureFlags(api *restapi.FireboxAPI) {
	// api.CommandLineOptionsGroups = []swag.CommandLineOptionsGroup{ ... }
}

func configureAPI(api *restapi.FireboxAPI) http.Handler {
	// configure the api here
	api.ServeError = errors.ServeError

	// Set your custom logger if needed. Default one is log.Printf
	// Expected interface func(string, ...interface{})
	//
	// Example:
	// api.Logger = log.Printf

	api.UseSwaggerUI()
	// To continue using redoc as your UI, uncomment the following line
	// api.UseRedoc()

	api.JSONConsumer = runtime.JSONConsumer()

	api.JSONProducer = runtime.JSONProducer()

	if api.VMPostVMRunHandler == nil {
		api.VMPostVMRunHandler = vm.PostVMRunHandlerFunc(func(params vm.PostVMRunParams) middleware.Responder {
			return middleware.NotImplemented("operation vm.PostVMRun has not yet been implemented")
		})
	}
	if api.ServiceInvokeHandler == nil {
		api.ServiceInvokeHandler = service.InvokeHandlerFunc(func(params service.InvokeParams) middleware.Responder {
			return middleware.NotImplemented("operation service.Invoke has not yet been implemented")
		})
	}
	if api.HealthIsHealthyHandler == nil {
		api.HealthIsHealthyHandler = health.IsHealthyHandlerFunc(func(params health.IsHealthyParams) middleware.Responder {
			return middleware.NotImplemented("operation health.IsHealthy has not yet been implemented")
		})
	}
	if api.HealthIsReadyHandler == nil {
		api.HealthIsReadyHandler = health.IsReadyHandlerFunc(func(params health.IsReadyParams) middleware.Responder {
			return middleware.NotImplemented("operation health.IsReady has not yet been implemented")
		})
	}

	api.PreServerShutdown = func() {
		StatusProber.SetNotReady(nil)
	}

	api.ServerShutdown = func() {
		// ServerShutdown is invoked only if all servers have successfully shut down
		StatusProber.SetNotHealthy(nil)
		// canceling server context
		serverCancel()
	}

	return setupGlobalMiddleware(api.Serve(setupMiddlewares))
}

// The TLS configuration before HTTPS server starts.
func configureTLS(tlsConfig *tls.Config) {
	// Make all necessary changes to the TLS configuration here.
}

// As soon as server is initialized but not run yet, this function will be called.
// If you need to modify a config, store server instance to stop it individually later, this is the place.
// This function can be called multiple times, depending on the number of serving schemes.
// scheme value will be set accordingly: "http", "https" or "unix".
func configureServer(s *http.Server, scheme, addr string) {
	s.BaseContext = func(_ net.Listener) context.Context {
		return ServerCtx
	}
}

// The middleware configuration is for the handler executors. These do not apply to the swagger.json document.
// The middleware executes after routing but before authentication, binding and validation.
func setupMiddlewares(handler http.Handler) http.Handler {
	return handler
}

// The middleware configuration happens before anything, this middleware also applies to serving the swagger.json document.
// So this is a good place to plug in a panic handling middleware, logging and metrics.
func setupGlobalMiddleware(handler http.Handler) http.Handler {
	return handler
}
