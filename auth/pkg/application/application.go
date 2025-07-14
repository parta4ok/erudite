package application

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/parta4ok/kvs/auth/internal/adapter/config"
	jwtprovider "github.com/parta4ok/kvs/auth/internal/adapter/jwt_provider"
	"github.com/parta4ok/kvs/auth/internal/adapter/storage/postgres"
	"github.com/parta4ok/kvs/auth/internal/cases"
	"github.com/parta4ok/kvs/auth/internal/cases/common"
	"github.com/parta4ok/kvs/auth/internal/entities"
	"github.com/parta4ok/kvs/auth/internal/port"
	"github.com/parta4ok/kvs/auth/internal/port/grpc/private"
	"github.com/parta4ok/kvs/auth/internal/port/http/public"
	"github.com/pkg/errors"
)

type App struct {
	CfgPath       string
	privateServer *private.Server
	publicServer  *public.Server
	cancel        context.CancelFunc
	wg            sync.WaitGroup
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

	storage := app.initStorage(cfg)

	provider := app.initJWTProvider(cfg)

	commandFactory := app.initCommandFactory(storage, provider)

	server := app.initPrivateGRPCPort(cfg, commandFactory)
	app.privateServer = server

	publicServer := app.initPublicHTTPPort(cfg, commandFactory)
	app.publicServer = publicServer
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

func (app *App) initStorage(cfg *config.Config) common.Storage {
	slog.Info("init storage started")

	var storage common.Storage

	storageType := cfg.GetServiceStorageType()
	connStr := cfg.GetStorageConnStr(storageType)
	switch storageType {
	case "postgres":
		s, err := postgres.NewStorage(connStr)
		if err != nil {
			app.panic(err)
		}
		storage = s
	default:
		err := errors.Wrap(entities.ErrInvalidParam, "invalid storage type")
		app.panic(err)
	}

	return storage

}

func (app *App) initCommandFactory(storage common.Storage,
	provider common.JWTProvider) port.CommandFactory {
	factory, err := cases.NewCommandFactory(cases.WithStorage(storage),
		cases.WithJWTProvider(provider))
	if err != nil {
		err := errors.Wrap(err, "new command factory init failure")
		app.panic(err)
	}

	return factory
}

func (app *App) initJWTProvider(cfg *config.Config) common.JWTProvider {
	slog.Info("init JWT provider")
	secret := cfg.GetJWTSecret()
	aud := cfg.GetJWTAudience()
	iss := cfg.GetJWTIssuer()
	ttl := cfg.GetJWTTTL()

	provider, err := jwtprovider.NewProvider(secret, aud, iss, ttl)
	if err != nil {
		err := errors.Wrap(err, "new command factory init failure")
		app.panic(err)
	}

	return provider
}

func (app *App) initPublicHTTPPort(cfg *config.Config, factory port.CommandFactory) *public.Server {
	slog.Info("init public http port started")

	port := cfg.GetPublicPort()
	interval := cfg.GetPublicTimeout()

	server, err := public.New(
		public.WithFactory(factory),
		public.WithConfig(&public.ServerCfg{Port: port, Timeout: interval}),
	)
	if err != nil {
		err := errors.Wrap(err, "new public http port init failure")
		app.panic(err)
	}

	return server
}

func (app *App) initPrivateGRPCPort(cfg *config.Config,
	factory port.CommandFactory) *private.Server {
	slog.Info("init private grpc port started")

	port := cfg.GetPrivatePort()

	server, err := private.NewServer(
		private.WithFactory(factory),
		private.WithPort(port),
	)
	if err != nil {
		err := errors.Wrap(err, "new private grpc port init failure")
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
		slog.Info("Starting private server")
		app.privateServer.StartServer()
	}()

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

	if app.privateServer != nil {
		slog.Info("Stopping private server...")
		app.privateServer.Stop()
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
