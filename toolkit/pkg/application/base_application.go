package application

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/parta4ok/kvs/toolkit/pkg/config"
	"github.com/parta4ok/kvs/toolkit/pkg/logger"
	"github.com/parta4ok/kvs/toolkit/pkg/port"
	"github.com/parta4ok/kvs/toolkit/pkg/tracer"
)

type BaseApplication struct {
	cfg    config.BaseConfig
	logger *logger.BaseLogger
	ports  []port.BasePort
}

type Builder struct {
	cfg config.BaseConfig
	app *BaseApplication
}

func NewBuilder(cfg config.BaseConfig) *Builder {
	return &Builder{
		cfg: cfg,
		app: &BaseApplication{
			cfg: cfg,
		},
	}
}

func (b *Builder) WithLogger() *Builder {
	logger := logger.NewBaseLogger(
		b.cfg.LogLevel(),
		b.cfg.LogAddSource(),
		b.cfg.LogFormat(),
		b.cfg.ServiceName(),
		b.cfg.ServiceVersion(),
	)
	logger.InitConfiguredLogger()
	b.app.logger = logger

	return b
}

func (b *Builder) WithTracer() *Builder {
	tp := tracer.NewPort(
		b.cfg.ServiceName(),
		b.cfg.ServiceVersion(),
		b.cfg.TracingEndpoint(),
		b.cfg.TracingEnabled(),
	)

	b.app.ports = append(b.app.ports, tp)
	return b
}

func (b *Builder) WithPort(p port.BasePort) *Builder {
	b.app.ports = append(b.app.ports, p)
	return b
}

func (b *Builder) Build() *BaseApplication {
	return b.app
}

func (app *BaseApplication) RunAndAwait(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	errChan := make(chan error, len(app.ports))

	var wg sync.WaitGroup
	for _, p := range app.ports {
		wg.Go(func() {
			if err := p.Start(ctx); err != nil {
				errChan <- err
			}
		})
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	defer signal.Stop(sigChan)

	var runErr error
	select {
	case err := <-errChan:
		slog.Error("port failed, terminating service", "error", err)
		runErr = err
	case sig := <-sigChan:
		slog.Info("shutdown signal received", "signal", sig.String())
	case <-ctx.Done():
		slog.Info("context cancelled")
	}

	cancel()
	app.stopAll()
	wg.Wait()

	return runErr
}

func (app *BaseApplication) stopAll() {
	shutdownCtx, cancel := context.WithTimeout(context.Background(), app.cfg.ShutdownTimeout())
	defer cancel()

	var wg sync.WaitGroup
	for _, p := range app.ports {
		wg.Go(func() {
			if err := p.Stop(shutdownCtx); err != nil {
				slog.Error("port stop failed", "error", err)
			}
		})
	}
	
	wg.Wait()
}
