package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/combust-labs/firebox/pkg/log"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type EchoConfig struct {
	Host string
	Port int
}

var (
	echoConfig = new(EchoConfig)
)

// echoCmd represents the echo command
var echoCmd = &cobra.Command{
	Use:   "echo",
	Short: "Start echo server",
	Run: func(cmd *cobra.Command, args []string) {
		runEcho()
	},
}

func init() {
	rootCmd.AddCommand(echoCmd)

	echoCmd.Flags().StringVar(&echoConfig.Host, "server-host", "0.0.0.0", "the IP to listen on")
	echoCmd.Flags().IntVar(&echoConfig.Port, "server-port", 8080, "the port to listen on for insecure connections")
}

func runEcho() {
	logger := newLogger()

	stopChannel := make(chan os.Signal, 1)
	signal.Notify(stopChannel, syscall.SIGINT, syscall.SIGTERM)
	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		stopSignal := <-stopChannel
		logger.Infof("received signal %v", stopSignal)
		cancel()
	}()

	if err := serveEcho(ctx, logger); err != nil {
		logger.Infof("failed to serve %v", err)
	}
	logger.Info("server stopped")
}

func serveEcho(ctx context.Context, logger *log.Logger) (err error) {
	mux := http.NewServeMux()
	mux.Handle("/", echoHandlerFunc())

	addr := fmt.Sprintf("%s:%d", echoConfig.Host, echoConfig.Port)
	srv := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	go func() {
		logger.Infof("Serving echo server at http://%s", addr)
		if err = srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatalf("listen failed: %v", err)
		}
	}()

	<-ctx.Done()
	logger.Info("Shutting down...")

	ctxShutDown, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer func() {
		cancel()
	}()

	if err := srv.Shutdown(ctxShutDown); err != nil {
		err = errors.Wrap(err, "Server shutdown failed")
	}
	if err == http.ErrServerClosed {
		err = nil
	}
	return err
}

func echoHandlerFunc() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data, err := ioutil.ReadAll(r.Body)
		if err != nil {
			http.Error(w, fmt.Sprintf("error reading request body: %s", err), http.StatusInternalServerError)
			return
		}
		_ = r.Body.Close()
		resp := &echoResponse{
			Method:  r.Method,
			Headers: r.Header,
			URL:     r.URL.String(),
			Data:    string(data),
		}
		body, _ := json.Marshal(resp)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body)
	}
}

type echoResponse struct {
	Method  string      `json:"method"`
	Headers http.Header `json:"headers"`
	URL     string      `json:"url"`
	Data    string      `json:"data"`
}
