package application

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

type Port interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	Type() string
}

type BaseApplication struct {
	ports   []Port
	cancel  context.CancelFunc
	wg      sync.WaitGroup
	timeout time.Duration
}

func NewBaseApplication() *BaseApplication {
	return &BaseApplication{}
}

func (app *BaseApplication) SetPorts(ports []Port) {
	app.ports = ports
}

func (app *BaseApplication) SetTimeout(timeout time.Duration) {
	app.timeout = timeout
}

//nolint:gosec //ok
func (app *BaseApplication) StartPortsWithGracefulShutdown() {
	ctx, cancel := context.WithCancel(context.Background())
	app.cancel = cancel

	sigOSChan := make(chan os.Signal, 1)
	signal.Notify(sigOSChan, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)

	for _, port := range app.ports {
		app.wg.Go(
			func() {
				slog.Info("starting listener", "type", port.Type())
				port.Start(ctx) //nolint:errcheck //ok
			})
	}

	select {
	case sig := <-sigOSChan:
		slog.Info("received os shutdown signal", "signal", sig.String())
		app.cancel()
		app.shutdown()
	case <-ctx.Done():
		slog.Info("application context cancelled")
		app.shutdown()
	}
}

//nolint:gosec //ok
func (app *BaseApplication) shutdown() {
	slog.Info("Starting graceful shutdown...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), app.timeout)
	defer shutdownCancel()

	for _, port := range app.ports {
		app.wg.Go(
			func() {
				slog.Info("stopping port", "port", port.Type())
				port.Stop(shutdownCtx) //nolint:errcheck //ok
			})
	}

	done := make(chan struct{})
	go func() {
		app.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		slog.Info("all ports stopped gracefully")
	case <-shutdownCtx.Done():
		slog.Warn("gracefull shutdown timeout exceeded, forcing exit")
	}

	if app.cancel != nil {
		app.cancel()
	}

	slog.Info("aplication shutdown completed")
}
