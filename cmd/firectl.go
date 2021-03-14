package cmd

import (
	"context"
	"github.com/combust-labs/firebox/pkg/log"
	"github.com/combust-labs/firebox/pkg/vmm"
	"github.com/oklog/run"
	"github.com/spf13/cobra"
	"os"
	"os/signal"
	"syscall"
)

// firectlCmd represents the firectl command
var firectlCmd = &cobra.Command{
	Use:   "firectl",
	Short: "Run single Firecracker MicroVM",
	Run: func(cmd *cobra.Command, args []string) {
		firectlRun()
	},
}

func init() {
	rootCmd.AddCommand(firectlCmd)
	initVMMConfigFlags(firectlCmd)
}

func firectlRun() {
	logger := newLogger()

	ctx := context.Background()
	shutdownCtx, shutdownFunc := context.WithCancel(ctx)
	defer shutdownFunc()

	var g run.Group
	{
		g.Add(func() error {
			logger.Infof("Install stop signal")
			waitForSignal(shutdownCtx, logger)
			return nil
		}, func(err error) {
			defer shutdownFunc()
			logger.Infof("Stop signal finished")
		})
	}
	{
		control := vmm.NewVMM(logger, *vmmConfig)
		g.Add(func() error {
			err := control.Start()
			if err != nil {
				return err
			}
			logger.Infof("VMM IP %v", control.GetIP())
			return control.WaitFinished()
		}, func(err error) {
			defer shutdownFunc()
			logger.Infof("VMM execution finished: %v", err)
			err = control.Stop()
			if err != nil {
				logger.Infof("VMM close error: %v", err)
			}
		})
	}
	logger.Infof("Finished with %v", g.Run())
}

func waitForSignal(ctx context.Context, logger *log.Logger) {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	select {
	case <-ctx.Done():
		logger.Infof("shutdown received ...")
	case q := <-quit:
		logger.Infof("quit %v received ... ", q)
	}
}
