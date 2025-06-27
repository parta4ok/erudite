package appication

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/parta4ok/kvs/knowledge_checker/internal/adapter/config"
	"github.com/parta4ok/kvs/knowledge_checker/internal/adapter/generator"
	"github.com/parta4ok/kvs/knowledge_checker/internal/adapter/storage/postgres"
	"github.com/parta4ok/kvs/knowledge_checker/internal/cases"
	"github.com/parta4ok/kvs/knowledge_checker/internal/entities"
	"github.com/parta4ok/kvs/knowledge_checker/internal/port/http/public"
	"github.com/pkg/errors"
)

type App struct {
	CfgPath      string
	publicServer *public.Server
	cancel       context.CancelFunc
	wg           sync.WaitGroup
}

func NewApp(cfgPath string) *App {
	return &App{
		CfgPath: cfgPath,
	}
}

func (app *App) Start() {
	cfg, err := config.NewConfig(app.CfgPath)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	app.initConfiguredLogger(cfg)
	slog.Info("Logger configuration completed")

	storage, sessionStorage := app.initStorage(cfg)
	generator := app.initGenerator()

	service := app.initSessionService(storage, sessionStorage, generator)

	server := app.initPublicPort(cfg, service)
	app.publicServer = server

	app.startWithGracefulShutdown()
}

func (app *App) initConfiguredLogger(cfg *config.Config) {
	level := parseLogLevel(cfg.GetLogLevel())

	opts := &slog.HandlerOptions{
		Level:     level,
		AddSource: cfg.GetLogAddSource(),
	}

	var handler slog.Handler

	switch cfg.GetLogFormat() {
	case "text":
		handler = slog.NewTextHandler(os.Stdout, opts)
	default:
		handler = slog.NewJSONHandler(os.Stdout, opts)
	}

	logger := slog.New(handler).With(
		"service", cfg.GetServiceName(),
		"version", cfg.GetServiceVersion(),
	)

	slog.SetDefault(logger)

	slog.Info("Logger reconfigured from config",
		"level", cfg.GetLogLevel(),
		"format", cfg.GetLogFormat(),
		"add_source", cfg.GetLogAddSource())
}

func parseLogLevel(levelStr string) slog.Level {
	switch levelStr {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func (app *App) initStorage(cfg *config.Config) (cases.Storage, entities.SessionStorage) {
	slog.Info("init storage started")

	var storage cases.Storage
	var sessionStorage entities.SessionStorage

	storageType := cfg.GetServiceStorageType()
	connStr := cfg.GetStorageConnStr(storageType)
	switch storageType {
	case "postgres":
		s, err := postgres.NewStorage(connStr)
		if err != nil {
			app.panic(err)
		}
		storage = s
		sessionStorage = s
	default:
		err := errors.Wrap(entities.ErrInvalidParam, "invalid storage type")
		app.panic(err)
	}

	return storage, sessionStorage

}

func (app *App) initGenerator() entities.IDGenerator {
	slog.Info("init generator started")
	var gen entities.IDGenerator
	g := generator.NewUint64Generator()
	gen = g

	return gen
}

func (app *App) initSessionService(storage cases.Storage, sessionStorage entities.SessionStorage,
	generator entities.IDGenerator) *cases.SessionService {
	slog.Info("init session_service started")

	serv, err := cases.NewSessionService(storage, sessionStorage, generator)
	if err != nil {
		err := errors.Wrap(err, "NewSessionService")
		app.panic(err)
	}

	return serv
}

func (app *App) initPublicPort(cfg *config.Config, sessionService public.Service) *public.Server {
	slog.Info("init public port started")

	port := cfg.GetPublicPort()

	server, err := public.New(
		public.WithService(sessionService),
		public.WithConfig(&public.ServerCfg{
			Port: port,
		}))
	if err != nil {
		err := errors.Wrap(err, "new public port init failure")
		app.panic(err)
	}

	return server
}

func (app *App) startWithGracefulShutdown() {
	ctx, cancel := context.WithCancel(context.Background())
	app.cancel = cancel

	sigOSChan := make(chan os.Signal, 1)
	signal.Notify(sigOSChan, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)

	app.wg.Add(1)
	go func() {
		defer app.wg.Done()
		slog.Info("Starting public server")
		app.publicServer.Start()
	}()

	select {
	case sig := <-sigOSChan:
		slog.Info("Received os shutdown signal", "signal", sig.String())
		app.shutdown()
	case <-ctx.Done():
		slog.Info("Application context cancelled")
		app.shutdown()
	}
}

func (app *App) shutdown() {
	slog.Info("Starting graceful shutdown...")

	shutdownTimeout := 2 * time.Second
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer shutdownCancel()

	if app.publicServer != nil {
		slog.Info("Stopping public server...")
		app.publicServer.Stop()
	}

	done := make(chan struct{})
	go func() {
		app.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		slog.Info("All services stopped gracefully")
	case <-shutdownCtx.Done():
		slog.Warn("Graceful shutdown timeout exceeded, forcing exit")
	}

	if app.cancel != nil {
		app.cancel()
	}

	slog.Info("Application shutdown completed")
}

func (app *App) Stop() {
	if app.cancel != nil {
		app.cancel()
	}
}

func (app *App) panic(err error, args ...any) {
	slog.Error(err.Error(), args...)
	os.Exit(1)
}
