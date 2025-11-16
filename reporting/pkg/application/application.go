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

	authservice "github.com/parta4ok/kvs/reporting/internal/adapter/auth_service"
	"github.com/parta4ok/kvs/reporting/internal/adapter/config"
	"github.com/parta4ok/kvs/reporting/internal/adapter/message_broker/nats"
	questionservice "github.com/parta4ok/kvs/reporting/internal/adapter/question_service"
	"github.com/parta4ok/kvs/reporting/internal/adapter/representer"
	"github.com/parta4ok/kvs/reporting/internal/cases"
	"github.com/parta4ok/kvs/reporting/internal/entities"
	"github.com/parta4ok/kvs/reporting/internal/port"
	"github.com/parta4ok/kvs/reporting/internal/port/http/public"
	consumer "github.com/parta4ok/kvs/reporting/internal/port/nats"
	"github.com/parta4ok/kvs/toolkit/pkg/accessor"
	"github.com/parta4ok/kvs/toolkit/pkg/broker/nats/publisher"

	"github.com/pkg/errors"
)

type App struct {
	CfgPath         string
	cfg             *config.Config
	publicServer    *public.Server
	consumer        *consumer.NatsConsumer
	cancel          context.CancelFunc
	wg              sync.WaitGroup
	pub             cases.MessageBroker
	authService     cases.AuthClient
	questionService cases.QuestionClient
	representer     entities.Representer
	accessor        public.Accessor
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

	app.cfg = cfg

	app.initConfiguredLogger(cfg)
	slog.Info("Logger configuration completed")

	app.initBroker()
	app.initAuthClient()
	app.initQuestionClient()
	app.initRepresenter()
	app.initAccessor()

	service := app.initReportingService()

	pubicServer := app.initPublicPort(service)
	app.publicServer = pubicServer
	consumer := app.initConsumer(service)
	app.consumer = consumer

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

func (app *App) initAccessor() {
	slog.Info("initAccessor started")

	a, err := accessor.NewRightAccessor()
	if err != nil {
		err := errors.Wrap(err, "new right accessor failure")
		app.panic(err)
	}

	app.accessor = a
}

func (app *App) initPublicPort(service port.Service) *public.Server {
	slog.Info("init public port started")

	port := app.cfg.GetPublicPort()
	timeout := app.cfg.GetPublicTimeout()

	server, err := public.New(
		public.WithService(service),
		public.WithIntrospector(app.authService),
		public.WithConfig(&public.ServerCfg{
			Port:    port,
			Timeout: timeout,
		}),
		public.WithAccessor(app.accessor))
	if err != nil {
		err := errors.Wrap(err, "new public port init failure")
		app.panic(err)
	}

	return server
}

func (app *App) initConsumer(service port.Service) *consumer.NatsConsumer {
	slog.Info("init consumer started")

	consumer, err := consumer.NewNatsConsumer(
		app.cfg.GetNatsURL(),
		app.cfg.GetNatsSubject(),
		service,
	)
	if err != nil {
		app.panic(err)
	}

	return consumer
}

func (app *App) initBroker() {
	slog.Info("init broker publisher started")

	pub, err := publisher.NewPublisher(app.cfg.GetNatsURL())
	if err != nil {
		app.panic(err)
	}

	b, err := nats.NewPublisher(pub)
	if err != nil {
		app.panic(err)
	}

	app.pub = b
}

func (app *App) initAuthClient() {
	slog.Info("init auth client started")

	authAdapter, err := authservice.NewAuthService(app.cfg.GetAuthConn())
	if err != nil {
		app.panic(err)
	}

	app.authService = authAdapter
}

func (app *App) initQuestionClient() {
	slog.Info("init question client started")

	client, err := questionservice.New(app.cfg.GetQuestionConn())
	if err != nil {
		app.panic(err)
	}

	app.questionService = client
}

func (app *App) initRepresenter() {
	slog.Info("init representer started")

	representer, err := representer.NewRepresenter()
	if err != nil {
		app.panic(err)
	}

	app.representer = representer
}

func (app *App) initReportingService() port.Service {
	var service port.Service

	serv, err := cases.NewReportingService(
		app.pub,
		app.representer,
		app.cfg.GetRepresenterFormat(),
		app.authService,
		app.questionService,
		app.cfg.GetWorkersLimit(),
	)

	if err != nil {
		app.panic(err)
	}

	service = serv

	return service
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

	app.wg.Add(1)
	go func() {
		defer app.wg.Done()
		slog.Info("Starting consumer")
		app.consumer.Start() //nolint:errcheck,gosec //ok
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

	if app.consumer != nil {
		slog.Info("Stopping consumer...")
		app.consumer.Stop() //nolint:errcheck,gosec //ok
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
