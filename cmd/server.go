package cmd

import (
	"context"
	"time"

	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/combust-labs/firebox/api/server"
	"github.com/combust-labs/firebox/api/server/restapi"
	"github.com/combust-labs/firebox/cmd/handlers"
	"github.com/combust-labs/firebox/pkg/actors/manager"
	"github.com/combust-labs/firebox/pkg/flags"
	"github.com/combust-labs/firebox/pkg/log"
	"github.com/combust-labs/firebox/pkg/prober"
	localprober "github.com/combust-labs/firebox/pkg/prober/local"
	"github.com/combust-labs/firebox/pkg/utils"
	"github.com/go-openapi/loads"
	"github.com/pkg/errors"

	"github.com/spf13/cobra"
)

type ServerConfig struct {
	server.Server
}

var (
	serverConfig = new(ServerConfig)
)

var serverCmd = &cobra.Command{
	Use:     "server",
	Short:   "Starts the server and serves the HTTP REST API",
	Aliases: []string{"serve"},
	Run: func(cmd *cobra.Command, args []string) {
		runServer()
	},
}

func init() {
	rootCmd.AddCommand(serverCmd)

	serverFlags := flags.ExtendFlagSet(serverCmd.Flags())

	serverFlags.StringSliceVar(&serverConfig.EnabledListeners, "server-scheme", []string{"http"}, "the listeners to enable, this can be repeated and defaults to the schemes in the swagger spec")
	serverFlags.DurationVar(&serverConfig.CleanupTimeout, "server-cleanup-timeout", 10*time.Second, "grace period for which to wait before killing idle connections")
	serverFlags.DurationVar(&serverConfig.GracefulTimeout, "server-graceful-timeout", 15*time.Second, "grace period for which to wait before shutting down the server")
	serverFlags.ByteSizeVar(&serverConfig.MaxHeaderSize, "server-max-header-size", 1048576, "controls the maximum number of bytes the server will read parsing the request header's keys and values, including the request line. It does not limit the size of the request body.")

	serverFlags.FilenameVar(&serverConfig.SocketPath, "server-socket-path", "/var/run/firebox.sock", "the unix socket to listen on")

	serverFlags.StringVar(&serverConfig.Host, "server-host", "localhost", "the IP to listen on")
	serverFlags.IntVar(&serverConfig.Port, "server-port", 0, "the port to listen on for insecure connections, defaults to a random value")
	serverFlags.IntVar(&serverConfig.ListenLimit, "server-listen-limit", 0, "limit the number of outstanding requests")

	serverFlags.DurationVar(&serverConfig.KeepAlive, "server-keep-alive", 3*time.Minute, "sets the TCP keep-alive timeouts on accepted connections. It prunes dead TCP connections")
	serverFlags.DurationVar(&serverConfig.ReadTimeout, "server-read-timeout", 30*time.Second, "maximum duration before timing out read of the request")
	serverFlags.DurationVar(&serverConfig.WriteTimeout, "server-write-timeout", 60*time.Second, "maximum duration before timing out write of the response")

	serverFlags.StringVar(&serverConfig.TLSHost, "server-tls-host", "", "the IP to listen on for tls, when not specified it'sc the same as --host")
	serverFlags.IntVar(&serverConfig.TLSPort, "server-tls-port", 0, "the port to listen on for secure connections, defaults to a random value")

	serverFlags.FilenameVar(&serverConfig.TLSCertificate, "server-tls-certificate", "", "the certificate to use for secure connections")
	serverFlags.FilenameVar(&serverConfig.TLSCertificateKey, "server-tls-key", "", "the private key to use for secure connections")
	serverFlags.FilenameVar(&serverConfig.TLSCACertificate, "server-tls-ca", "", "he certificate authority file to be used with mutual tls auth")

	serverFlags.DurationVar(&serverConfig.TLSKeepAlive, "server-tls-keep-alive", 0, "sets the TCP keep-alive timeouts on accepted connections. It prunes dead TCP connections")
	serverFlags.DurationVar(&serverConfig.TLSReadTimeout, "server-tls-read-timeout", 0, "maximum duration before timing out read of the request")
	serverFlags.DurationVar(&serverConfig.TLSWriteTimeout, "server-tls-write-timeout", 0, "maximum duration before timing out write of the response")

	initVMMConfigFlags(serverCmd)
}

type Server struct {
	*server.Server

	ctx    context.Context
	cancel context.CancelFunc

	logger *log.Logger

	httpProber   *localprober.HTTPProbe
	statusProber prober.Probe

	system *actor.ActorSystem
	defers utils.Defers
}

func NewServer(ctx context.Context, logger *log.Logger) (*Server, error) {
	sCtx, cancel := context.WithCancel(ctx)

	srv := &Server{
		Server:       server.NewServer(nil),
		ctx:          sCtx,
		cancel:       cancel,
		logger:       logger,
		httpProber:   server.HTTPProber,
		statusProber: server.StatusProber,
		system:       actor.NewActorSystem(),
		defers:       utils.NewDefers(),
	}
	srv.configureServerFlags()
	api, err := srv.instantiateAPI()
	if err != nil {
		return nil, err
	}
	srv.SetAPI(api)
	return srv, nil
}

func (s *Server) configureServerFlags() {
	s.EnabledListeners = serverConfig.EnabledListeners
	s.CleanupTimeout = serverConfig.CleanupTimeout
	s.GracefulTimeout = serverConfig.GracefulTimeout
	s.MaxHeaderSize = serverConfig.MaxHeaderSize
	s.SocketPath = serverConfig.SocketPath

	s.Host = serverConfig.Host
	s.Port = serverConfig.Port
	s.ListenLimit = serverConfig.ListenLimit
	s.KeepAlive = serverConfig.KeepAlive
	s.ReadTimeout = serverConfig.ReadTimeout
	s.WriteTimeout = serverConfig.WriteTimeout

	s.TLSHost = serverConfig.TLSHost
	s.TLSPort = serverConfig.TLSPort
	s.TLSCertificate = serverConfig.TLSCertificate
	s.TLSCertificateKey = serverConfig.TLSCertificateKey
	s.TLSCACertificate = serverConfig.TLSCACertificate
	s.TLSListenLimit = serverConfig.TLSListenLimit
	s.TLSKeepAlive = serverConfig.TLSKeepAlive
	s.TLSReadTimeout = serverConfig.TLSReadTimeout
	s.TLSWriteTimeout = serverConfig.TLSWriteTimeout
}

func runServer() {
	ctx := context.Background()

	logger := newLogger()
	srv, err := NewServer(ctx, logger)
	if err != nil {
		logger.WithError(err).Fatalf("preparing server failed")
	}
	defer srv.Shutdown()
	defer srv.defers.Exec()

	srv.statusProber.SetHealthy()
	srv.statusProber.SetReady()
	if err := srv.Serve(); err != nil {
		logger.WithError(err).Fatalf("serve failed")
	}
}

func (s *Server) instantiateAPI() (*restapi.FireboxAPI, error) {
	swaggerSpec, err := loads.Embedded(server.SwaggerJSON, server.FlatSwaggerJSON)
	if err != nil {
		return nil, errors.Wrap(err, "loading swagger api failed")
	}
	api := restapi.NewFireboxAPI(swaggerSpec)
	api.Logger = func(format string, args ...interface{}) {
		s.logger.Printf(format, args...)
	}

	api.HealthIsHealthyHandler = s.httpProber.HealthyHandler()
	api.HealthIsReadyHandler = s.httpProber.ReadyHandler()

	mgr := manager.NewVMMManager(s.logger, *vmmConfig)
	mgr.Init(s.system)
	s.defers.Add(func() {
		_ = mgr.Close()
	})
	api.VMPostVMRunHandler = handlers.NewVMPostVMRunHandler(s.logger, mgr)
	api.ServiceInvokeHandler = handlers.NewServiceInvokeHandler(s.logger, mgr)
	return api, nil
}
