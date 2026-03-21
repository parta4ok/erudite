package application

import (
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/parta4ok/kvs/auth/internal/adapter/config"
	"github.com/parta4ok/kvs/auth/internal/adapter/generator/google"
	"github.com/parta4ok/kvs/auth/internal/adapter/hasher/bcryption"
	jwtprovider "github.com/parta4ok/kvs/auth/internal/adapter/jwt_provider"
	"github.com/parta4ok/kvs/auth/internal/adapter/message_broker/nats"
	"github.com/parta4ok/kvs/auth/internal/adapter/storage/postgres"
	"github.com/parta4ok/kvs/auth/internal/cases"
	"github.com/parta4ok/kvs/auth/internal/cases/common"
	"github.com/parta4ok/kvs/auth/internal/entities"
	"github.com/parta4ok/kvs/auth/internal/port"
	"github.com/parta4ok/kvs/auth/internal/port/grpc/private"
	"github.com/parta4ok/kvs/auth/internal/port/http/public"
	"github.com/parta4ok/kvs/toolkit/pkg/accessor"
	baseApplication "github.com/parta4ok/kvs/toolkit/pkg/application"
	"github.com/parta4ok/kvs/toolkit/pkg/broker/nats/publisher"
	"github.com/parta4ok/kvs/toolkit/pkg/logger"
	"github.com/parta4ok/kvs/toolkit/pkg/tracing"
	projectTracer "github.com/parta4ok/kvs/toolkit/pkg/tracing/jaeger"
	otelTracer "github.com/parta4ok/kvs/toolkit/pkg/tracing/otel"

	"github.com/pkg/errors"
)

type App struct {
	cfg           *config.Config
	privateServer *private.Server
	publicServer  *public.Server
	tracerPort    *tracing.TracerPort
	tracing       bool
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
	baseLogger := logger.NewBaseLogger(
		app.cfg.GetLogLevel(),
		app.cfg.GetLogAddSource(),
		app.cfg.GetLogFormat(),
		app.cfg.GetServiceName(),
		app.cfg.GetServiceVersion(),
	)
	baseLogger.InitConfiguredLogger()

	tracerPort := app.initTracer()
	app.tracerPort = tracerPort

	storage := app.initStorage()
	provider := app.initJWTProvider()
	hasher := app.initHasher()
	generator := app.initGenerator()
	accessor := app.initAccessor()
	messageBroker := app.initBroker()

	commandFactory := app.initCommandFactory(storage, provider, hasher, generator, messageBroker)

	privateServer := app.initPrivateGRPCPort(commandFactory)
	app.privateServer = privateServer

	publicServer := app.initPublicHTTPPort(commandFactory, accessor)
	app.publicServer = publicServer

	baseApp := baseApplication.NewBaseApplication()
	baseApp.SetTimeout(5 * time.Second)
	baseApp.SetPorts([]baseApplication.Port{
		app.privateServer,
		app.publicServer,
		app.tracerPort,
	})

	baseApp.StartPortsWithGracefulShutdown()
}

func (app *App) initBroker() common.MessageBroker {
	slog.Info("init broker started")

	var broker common.MessageBroker

	pub := app.initNatsPub()
	nats, err := nats.NewPublisher(pub)
	if err != nil {
		app.panic(err)
	}

	broker = nats

	return broker
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
		serviceURL := app.cfg.GetTracingInfraURL(systemName)
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

func (app *App) initAccessor() public.Accessor {
	slog.Info("initAccessor started")

	a, err := accessor.NewRightAccessor()
	if err != nil {
		err := errors.Wrap(err, "new right accessor failure")
		app.panic(err)
	}

	return a
}

func (app *App) initStorage() common.Storage {
	slog.Info("init storage started")

	var storage common.Storage

	storageType := app.cfg.GetServiceStorageType()
	connStr := app.cfg.GetStorageConnStr(storageType)
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

func (app *App) initHasher() common.Hasher {
	slog.Info("init hasher started")

	h, err := bcryption.NewHasher()
	if err != nil {
		err := errors.Wrap(err, "hasher init failure")
		app.panic(err)
	}

	return h
}

func (app *App) initGenerator() common.IDGenerator {
	slog.Info("init generator started")

	g, err := google.NewGenerator()
	if err != nil {
		err := errors.Wrap(err, "generator init failure")
		app.panic(err)
	}

	return g
}

func (app *App) initCommandFactory(storage common.Storage,
	provider common.JWTProvider, hasher common.Hasher,
	generator common.IDGenerator, messageBroker common.MessageBroker) port.CommandFactory {
	slog.Info("initCommandFactory started")

	factory, err := cases.NewCommandFactory(
		cases.WithStorage(storage),
		cases.WithJWTProvider(provider),
		cases.WithHasher(hasher),
		cases.WithIDGenerator(generator),
		cases.WithMessageBroker(messageBroker),
	)
	if err != nil {
		err := errors.Wrap(err, "new command factory init failure")
		app.panic(err)
	}

	return factory
}

func (app *App) initJWTProvider() common.JWTProvider {
	slog.Info("init JWT provider")
	secret := app.cfg.GetJWTSecret()
	aud := app.cfg.GetJWTAudience()
	iss := app.cfg.GetJWTIssuer()
	ttl := app.cfg.GetJWTTTL()

	provider, err := jwtprovider.NewProvider(secret, aud, iss, ttl)
	if err != nil {
		err := errors.Wrap(err, "new JWT provider init failure")
		app.panic(err)
	}

	return provider
}

func (app *App) initPublicHTTPPort(factory port.CommandFactory,
	accessor public.Accessor) *public.Server {
	slog.Info("init public http port started")

	port := app.cfg.GetPublicPort()
	interval := app.cfg.GetPublicTimeout()

	server, err := public.New(
		public.WithFactory(factory),
		public.WithAccessor(accessor),
		public.WithConfig(&public.ServerCfg{Port: port, Timeout: interval}),
	)
	if err != nil {
		err := errors.Wrap(err, "new public http port init failure")
		app.panic(err)
	}

	return server
}

func (app *App) initPrivateGRPCPort(factory port.CommandFactory) *private.Server {
	slog.Info("init private grpc port started")

	port := app.cfg.GetPrivatePort()

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

func (app *App) initNatsPub() *publisher.Publisher {
	slog.Info("init nats publisher started")

	natsUrl := app.cfg.GetNatsURL()
	pub, err := publisher.NewPublisher(natsUrl)
	if err != nil {
		app.panic(err)
	}

	return pub
}

func (app *App) panic(err error, args ...any) {
	slog.Error(err.Error(), args...)
	os.Exit(1)
}
