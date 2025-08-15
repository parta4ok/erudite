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

	"github.com/parta4ok/kvs/question/internal/adapter/config"
	cryptoprocessing "github.com/parta4ok/kvs/question/internal/adapter/generator/crypto_processing"
	authservice "github.com/parta4ok/kvs/question/internal/adapter/introspector/auth_service"
	"github.com/parta4ok/kvs/question/internal/adapter/message_broker/nats"
	"github.com/parta4ok/kvs/question/internal/adapter/storage/postgres"
	"github.com/parta4ok/kvs/question/internal/cases"
	"github.com/parta4ok/kvs/question/internal/entities"
	"github.com/parta4ok/kvs/question/internal/port/http/public"
	"github.com/parta4ok/kvs/toolkit/pkg/accessor"
	"github.com/parta4ok/kvs/toolkit/pkg/broker/nats/publisher"
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
	authClient := app.initAuthServiceClient(cfg)
	accessor := app.initAccessor(cfg)

	service := app.initSessionServiceBase(storage, sessionStorage, generator)
	broker := app.initBroker(cfg)

	wrappedService := app.initWrappedSessionService(cfg, service, broker)

	server := app.initPublicPort(cfg, wrappedService, authClient, accessor)
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

func (app *App) initBroker(cfg *config.Config) cases.MessageBroker {
	slog.Info("init broker started")

	var broker cases.MessageBroker
	subject := cfg.GetNatsSubject()

	pub := app.initNatsPub(cfg)
	nats, err := nats.NewPublisher(pub, subject)
	if err != nil {
		app.panic(err)
	}

	broker = nats

	return broker
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

func (app *App) initAccessor(_ *config.Config) public.Accessor {
	slog.Info("initAccessor started")
	var acessor public.Accessor

	a, err := accessor.NewRightAccessor()
	if err != nil {
		err := errors.Wrap(err, "new right accessor failure")
		app.panic(err)
	}

	acessor = a

	return acessor
}

func (app *App) initGenerator() entities.IDGenerator {
	slog.Info("init generator started")
	var gen entities.IDGenerator
	g := cryptoprocessing.NewUint64Generator()
	gen = g

	return gen
}

func (app *App) initNatsPub(cfg *config.Config) *publisher.Publisher {
	slog.Info("init nats publisher started")

	natsUrl := cfg.GetNatsURL()
	pub, err := publisher.NewPublisher(natsUrl)
	if err != nil {
		app.panic(err)
	}

	return pub
}

func (app *App) initSessionServiceBase(storage cases.Storage,
	sessionStorage entities.SessionStorage,
	generator entities.IDGenerator) cases.SessionService {
	slog.Info("init session_service started")

	var sessionService cases.SessionService

	serv, err := cases.NewSessionServiceBase(storage, sessionStorage, generator)
	if err != nil {
		err := errors.Wrap(err, "NewSessionServiceBase")
		app.panic(err)
	}

	sessionService = serv

	return sessionService
}

func (app *App) initWrappedSessionService(cfg *config.Config, service cases.SessionService,
	broker cases.MessageBroker) cases.SessionService {
	slog.Info("init wrapped_session_service started")
	var wrappedService cases.SessionService

	brokerEventTimeOut := cfg.GetEventTimeout()
	srv, err := cases.NewSessionServiceBusDecorator(service, broker,
		cases.WithCustomEventTimeout(brokerEventTimeOut))
	if err != nil {
		app.panic(err)
	}

	wrappedService = srv

	return wrappedService
}

func (app *App) initAuthServiceClient(cfg *config.Config) public.Introspector {
	slog.Info("init auth service client started")

	var authClient public.Introspector

	addr := cfg.GetAuthConn()
	if addr == "" {
		err := errors.Wrap(entities.ErrInvalidParam, "get auth address failure")
		app.panic(err)
	}

	client, err := authservice.NewAuthService(addr)
	if err != nil {
		err := errors.Wrap(entities.ErrInvalidParam, "new auth service client failure")
		app.panic(err)
	}

	authClient = client

	return authClient
}

func (app *App) initPublicPort(cfg *config.Config, sessionServiceBase cases.SessionService,
	authClient public.Introspector, accessor public.Accessor) *public.Server {
	slog.Info("init public port started")

	port := cfg.GetPublicPort()
	timeout := cfg.GetPublicTimeout()

	server, err := public.New(
		public.WithService(sessionServiceBase),
		public.WithIntrospector(authClient),
		public.WithConfig(&public.ServerCfg{
			Port:    port,
			Timeout: timeout,
		}),
		public.WithAccessor(accessor))
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
