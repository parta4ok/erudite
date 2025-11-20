package appication

import (
	"fmt"
	"log/slog"
	"os"
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
	baseApplication "github.com/parta4ok/kvs/toolkit/pkg/application"
	"github.com/parta4ok/kvs/toolkit/pkg/broker/nats/publisher"
	"github.com/parta4ok/kvs/toolkit/pkg/tracing"
	projectTracer "github.com/parta4ok/kvs/toolkit/pkg/tracing/jaeger"
	otelTracer "github.com/parta4ok/kvs/toolkit/pkg/tracing/otel"

	"github.com/pkg/errors"
)

type App struct {
	cfg             *config.Config
	tracingPort     *tracing.TracerPort
	publicServer    *public.Server
	consumer        *consumer.NatsConsumer
	pub             cases.MessageBroker
	authService     cases.AuthClient
	questionService cases.QuestionClient
	representer     entities.Representer
	accessor        public.Accessor
	tracing         bool
}

func NewApp(cfgPath string) *App {
	cfg, err := config.NewConfig(cfgPath)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	return &App{
		cfg: cfg,
	}
}

func (app *App) Start() {

	app.initConfiguredLogger()
	slog.Info("logger configuration completed")

	tracingPort := app.initTracer()
	app.tracingPort = tracingPort

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

	baseApp := baseApplication.NewBaseApplication()
	baseApp.SetTimeout(5 * time.Second)
	baseApp.SetPorts([]baseApplication.Port{
		app.publicServer,
		app.consumer,
		app.tracingPort,
	})

	baseApp.StartPortsWithGracefulShutdown()
}

func (app *App) initConfiguredLogger() {
	level := parseLogLevel(app.cfg.GetLogLevel())

	opts := &slog.HandlerOptions{
		Level:     level,
		AddSource: app.cfg.GetLogAddSource(),
	}

	var handler slog.Handler

	switch app.cfg.GetLogFormat() {
	case "text":
		handler = slog.NewTextHandler(os.Stdout, opts)
	default:
		handler = slog.NewJSONHandler(os.Stdout, opts)
	}

	logger := slog.New(handler).With(
		"service", app.cfg.GetServiceName(),
		"version", app.cfg.GetServiceVersion(),
	)

	slog.SetDefault(logger)

	slog.Info("Logger reconfigured from config",
		"level", app.cfg.GetLogLevel(),
		"format", app.cfg.GetLogFormat(),
		"add_source", app.cfg.GetLogAddSource())
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

func (app *App) initTracer() *tracing.TracerPort {
	slog.Info("init tracer started")

	if !app.cfg.IsTracingEnabled() {
		slog.Info("tracing disabled")
		tracerPort := tracing.NewTracePort(&tracing.NoOpTracer{})
		return tracerPort
	}

	app.tracing = true
	systemName := app.cfg.GetTracingType()
	serviceName := app.cfg.TracingSystemName()

	var tracer tracing.Tracer
	var err error

	switch systemName {
	case "otel", "opentelemetry":
		endpoint := app.cfg.GetOtelEndpoint()
		tracer, err = otelTracer.NewOtelTracer(serviceName, endpoint)
		if err != nil {
			app.panic(errors.Wrapf(err, "otel tracer init failure: %v", err))
		}
		slog.Info("tracer initialized successfully",
			slog.String("type", "opentelemetry"),
			slog.String("service", serviceName),
			slog.String("endpoint", endpoint),
		)
	case "jaeger":
		serviceURL := app.cfg.GetTracingInfraURL(systemName)
		tracer, err = projectTracer.NewJaegerTracer(serviceName, serviceURL)
		if err != nil {
			app.panic(errors.Wrapf(err, "jaeger tracer init failure: %v", err))
		}
		slog.Info("tracer initialized successfully",
			slog.String("type", systemName),
			slog.String("service", serviceName),
			slog.String("endpoint", serviceURL),
		)
	default:
		app.panic(errors.Wrap(entities.ErrInvalidParam, "unsupported tracing system: "+systemName))
	}

	tracerPort := tracing.NewTracePort(tracer)
	return tracerPort
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
		app.cfg.GetAsyncTimeout(),
	)

	if err != nil {
		app.panic(err)
	}

	service = serv

	return service
}

func (app *App) panic(err error, args ...any) {
	slog.Error(err.Error(), args...)
	os.Exit(1)
}
