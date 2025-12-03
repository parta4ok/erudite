package appication

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/parta4ok/kvs/question/internal/adapter/config"
	cryptoprocessing "github.com/parta4ok/kvs/question/internal/adapter/generator/crypto_processing"
	authservice "github.com/parta4ok/kvs/question/internal/adapter/introspector/auth_service"
	"github.com/parta4ok/kvs/question/internal/adapter/message_broker/nats"
	"github.com/parta4ok/kvs/question/internal/adapter/storage"
	"github.com/parta4ok/kvs/question/internal/adapter/storage/postgres"
	"github.com/parta4ok/kvs/question/internal/cases"
	"github.com/parta4ok/kvs/question/internal/entities"
	"github.com/parta4ok/kvs/question/internal/port/http/private"
	"github.com/parta4ok/kvs/question/internal/port/http/public"
	"github.com/parta4ok/kvs/toolkit/pkg/accessor"
	baseApplication "github.com/parta4ok/kvs/toolkit/pkg/application"
	"github.com/parta4ok/kvs/toolkit/pkg/broker/nats/publisher"
	"github.com/parta4ok/kvs/toolkit/pkg/cron"
	"github.com/parta4ok/kvs/toolkit/pkg/logger"
	"github.com/parta4ok/kvs/toolkit/pkg/tracing"
	projectTracer "github.com/parta4ok/kvs/toolkit/pkg/tracing/jaeger"
	otelTracer "github.com/parta4ok/kvs/toolkit/pkg/tracing/otel"

	"github.com/pkg/errors"
)

type App struct {
	config        *config.Config
	publicServer  *public.Server
	privateServer *private.Server
	tracerPort    *tracing.TracerPort
	sheduler      *cron.Sheduler
	tracing       bool
}

func NewApp(cfgPath string) *App {
	cfg, err := config.NewConfig(cfgPath)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	return &App{
		config: cfg,
	}
}

func (app *App) Start() {
	baseLogger := logger.NewBaseLogger(
		app.config.GetLogLevel(),
		app.config.GetLogAddSource(),
		app.config.GetLogFormat(),
		app.config.GetServiceName(),
		app.config.GetServiceVersion(),
	)
	baseLogger.InitConfiguredLogger()
	slog.Info("logger configuration completed")

	tracerPort := app.initTracer()
	app.tracerPort = tracerPort

	storage, sessionStorage, eventStorage := app.initStorage()
	generator := app.initGenerator()
	authClient := app.initAuthServiceClient()
	accessor := app.initAccessor()

	service := app.initSessionServiceBase(storage, sessionStorage, generator)
	broker := app.initBroker()

	sheduler := app.initSheduler(eventStorage, broker)
	app.sheduler = sheduler

	pubicServer := app.initPublicPort(service, authClient, accessor)
	app.publicServer = pubicServer

	privateServer := app.initPrivatePort(service)
	app.privateServer = privateServer

	ports := []baseApplication.Port{
		app.privateServer,
		app.publicServer,
		app.tracerPort,
		app.sheduler,
	}

	baseApp := baseApplication.NewBaseApplication()
	baseApp.SetTimeout(4 * time.Second)
	baseApp.SetPorts(ports)
	baseApp.StartPortsWithGracefulShutdown()
}

func (app *App) initTracer() *tracing.TracerPort {
	slog.Info("init tracer started")

	if !app.config.IsTracingEnabled() {
		slog.Info("tracing disabled")
		tracerPort := tracing.NewTracePort(&tracing.NoOpTracer{})
		return tracerPort
	}

	app.tracing = true
	systemName := app.config.GetTracingType()
	serviceName := app.config.TracingSystemName()

	var tracer tracing.Tracer
	var err error

	switch systemName {
	case "otel", "opentelemetry":
		endpoint := app.config.GetOtelEndpoint()
		tracer, err = otelTracer.NewOtelTracerAdapter(serviceName, endpoint)
		if err != nil {
			app.panic(errors.Wrapf(err, "otel tracer init failure: %v", err))
		}
		slog.Info("tracer initialized successfully",
			slog.String("type", "opentelemetry"),
			slog.String("service", serviceName),
			slog.String("endpoint", endpoint),
		)
	case "jaeger":
		serviceURL := app.config.GetTracingInfraURL(systemName)
		tracer, err = projectTracer.NewJaegerTracerAdapter(serviceName, serviceURL)
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

func (app *App) initBroker() cases.MessageBroker {
	slog.Info("init broker started")

	var broker cases.MessageBroker
	subject := app.config.GetNatsSubject()

	pub := app.initNatsPub()
	nats, err := nats.NewPublisher(pub, subject)
	if err != nil {
		app.panic(err)
	}

	broker = nats

	return broker
}

func (app *App) initStorage() (cases.Storage, entities.SessionStorage, storage.EventStorage) {
	slog.Info("init storage started")

	var baseStorage cases.Storage
	var sessionStorage entities.SessionStorage
	var eventStorage storage.EventStorage

	storageType := app.config.GetServiceStorageType()
	connStr := app.config.GetStorageConnStr(storageType)
	switch storageType {
	case "postgres":
		s, err := postgres.NewStorage(connStr)
		if err != nil {
			app.panic(err)
		}
		baseStorage = s
		sessionStorage = s
		eventStorage = s
	default:
		err := errors.Wrap(entities.ErrInvalidParam, "invalid storage type")
		app.panic(err)
	}

	return baseStorage, sessionStorage, eventStorage
}

func (app *App) initAccessor() public.Accessor {
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

func (app *App) initNatsPub() *publisher.Publisher {
	slog.Info("init nats publisher started")

	natsUrl := app.config.GetNatsURL()
	pub, err := publisher.NewPublisher(natsUrl)
	if err != nil {
		app.panic(err)
	}

	return pub
}

func (app *App) initSessionServiceBase(
	storage cases.Storage,
	sessionStorage entities.SessionStorage,
	generator entities.IDGenerator) cases.SessionService {
	slog.Info("init session_service started")

	var sessionService cases.SessionService

	respondTime := app.config.GetTimeToRespond()

	serv, err := cases.NewSessionServiceBase(storage, sessionStorage, generator,
		cases.WithCustomRespondTime(respondTime))
	if err != nil {
		err := errors.Wrap(err, "NewSessionServiceBase")
		app.panic(err)
	}

	sessionService = serv

	return sessionService
}

func (app *App) initAuthServiceClient() public.Introspector {
	slog.Info("init auth service client started")

	var authClient public.Introspector

	addr := app.config.GetAuthConn()
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

func (app *App) initPublicPort(
	sessionServiceBase cases.SessionService,
	authClient public.Introspector,
	accessor public.Accessor,
) *public.Server {
	slog.Info("init public port started")

	port := app.config.GetPublicPort()
	timeout := app.config.GetPublicTimeout()
	dailyLimit := app.config.GetSessionLimit()

	server, err := public.New(
		public.WithCustomDailySessionLimit(dailyLimit),
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

func (app *App) initPrivatePort(sessionServiceBase cases.SessionService) *private.Server {
	slog.Info("init private port started")

	port := app.config.GetPrivatePort()
	timeout := app.config.GetPrivateTimeout()

	server, err := private.New(
		private.WithService(sessionServiceBase),
		private.WithConfig(&private.ServerCfg{
			Port:    port,
			Timeout: timeout,
		}),
	)
	if err != nil {
		err := errors.Wrap(err, "new private port init failure")
		app.panic(err)
	}

	return server
}

func (app *App) initSheduler(
	eventStorage storage.EventStorage,
	broker cases.MessageBroker,
) *cron.Sheduler {
	sheduler, err := cron.NewSheduler()
	if err != nil {
		app.panic(err)
	}

	if err = sheduler.NewJob(app.config.GetPublisherInterval(),
		app.publishEvents, eventStorage, broker); err != nil {
		app.panic(err)
	}
	if err = sheduler.NewJob(app.config.GetFlusherInterval(),
		app.flushEvents, eventStorage); err != nil {
		app.panic(err)
	}

	return sheduler
}

func (app *App) panic(err error, args ...any) {
	slog.Error(err.Error(), args...)
	os.Exit(1)
}

func (app *App) publishEvents(eventStorage storage.EventStorage, broker cases.MessageBroker) {
	slog.Info("cron publishEvents start")

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()

	unpublishedEvents, err := eventStorage.GetUnpublishedEvents(ctx)
	if err != nil {
		slog.Warn("get unpublished events", "error", err)
	}

	for _, event := range unpublishedEvents {
		fn := func(ctx context.Context) error {
			if err := broker.Publish(ctx, event); err != nil {
				slog.Warn("publish event", "error", err)
				return err
			}

			return nil
		}

		if err := eventStorage.MarkEventAsPublished(ctx, event.Num(), fn); err != nil {
			slog.Warn("mark event as published", "error", err)
		}
	}

	slog.Info("cron publishEvents finished")
}

func (app *App) flushEvents(eventStorage storage.EventStorage) {
	slog.Info("cron flushEvents start")

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()

	if err := eventStorage.FlushPublishedEvents(ctx); err != nil {
		slog.Warn("flush published events", "error", err)
	}
	slog.Info("cron flushEvents finished")
}
