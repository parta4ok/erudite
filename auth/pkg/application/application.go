package application

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/parta4ok/kvs/auth/internal/adapter/config"
	"github.com/parta4ok/kvs/auth/internal/adapter/generator/google"
	"github.com/parta4ok/kvs/auth/internal/adapter/hasher/bcryption"
	jwtprovider "github.com/parta4ok/kvs/auth/internal/adapter/jwt_provider"
	natsbroker "github.com/parta4ok/kvs/auth/internal/adapter/message_broker/nats"
	"github.com/parta4ok/kvs/auth/internal/adapter/storage/postgres"
	"github.com/parta4ok/kvs/auth/internal/cases"
	"github.com/parta4ok/kvs/auth/internal/cases/common"
	"github.com/parta4ok/kvs/auth/internal/entities"
	"github.com/parta4ok/kvs/auth/internal/port"
	grpcprivate "github.com/parta4ok/kvs/auth/internal/port/grpc/private"
	"github.com/parta4ok/kvs/auth/internal/port/http/public"
	"github.com/parta4ok/kvs/toolkit/pkg/accessor"
	baseApplication "github.com/parta4ok/kvs/toolkit/pkg/application"
	"github.com/parta4ok/kvs/toolkit/pkg/broker/nats/publisher"
	grpcport "github.com/parta4ok/kvs/toolkit/pkg/port/grpc"
	httpport "github.com/parta4ok/kvs/toolkit/pkg/port/http"

	"github.com/pkg/errors"
)

type App struct {
	cfg *config.Config
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
	storage := app.initStorage()
	provider := app.initJWTProvider()
	hasher := app.initHasher()
	generator := app.initGenerator()
	acc := app.initAccessor()
	messageBroker := app.initBroker()

	commandFactory := app.initCommandFactory(storage, provider, hasher, generator, messageBroker)

	privatePort := app.initPrivateGRPCPort(commandFactory)
	publicPort := app.initPublicHTTPPort(commandFactory, acc)

	baseApp := baseApplication.NewBuilder(app.cfg).
		WithLogger().
		WithTracer().
		WithPort(privatePort).
		WithPort(publicPort).
		Build()

	if err := baseApp.RunAndAwait(context.Background()); err != nil {
		app.panic(err)
	}
}

func (app *App) initBroker() common.MessageBroker {
	slog.Info("init broker started")

	pub := app.initNatsPub()
	broker, err := natsbroker.NewPublisher(pub)
	if err != nil {
		app.panic(err)
	}

	return broker
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
	acc public.Accessor) *httpport.Port {
	slog.Info("init public http port started")

	addr := app.cfg.GetPublicPort()
	timeout := app.cfg.GetPublicTimeout()

	server, err := public.New(
		public.WithFactory(factory),
		public.WithAccessor(acc),
	)
	if err != nil {
		err := errors.Wrap(err, "new public server init failure")
		app.panic(err)
	}

	httpPort, err := httpport.NewPort(
		httpport.Config{Addr: addr, Timeout: timeout},
		httpport.WithType(public.PortType),
		httpport.WithRoutes(server.Routes()...),
	)
	if err != nil {
		err := errors.Wrap(err, "new public http port init failure")
		app.panic(err)
	}

	return httpPort
}

func (app *App) initPrivateGRPCPort(factory port.CommandFactory) *grpcport.Port {
	slog.Info("init private grpc port started")

	grpcAddr := ":" + app.cfg.GetPrivatePort()

	authService, err := grpcprivate.New(
		grpcprivate.WithFactory(factory),
	)
	if err != nil {
		err := errors.Wrap(err, "new auth grpc service init failure")
		app.panic(err)
	}

	grpcPort, err := grpcport.NewPort(
		grpcport.Config{Addr: grpcAddr},
		grpcport.WithType(grpcprivate.PortType),
		grpcport.WithServerOptions(grpcprivate.ServerOptions()...),
		grpcport.WithRegister(authService.Register),
	)
	if err != nil {
		err := errors.Wrap(err, "new private grpc port init failure")
		app.panic(err)
	}

	return grpcPort
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
	slog.Error("application panic", append([]any{"error", err.Error()}, args...)...)
	os.Exit(1)
}
