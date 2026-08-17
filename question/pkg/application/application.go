package application

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/parta4ok/kvs/question/internal/adapter/config"
	cryptoprocessing "github.com/parta4ok/kvs/question/internal/adapter/generator/crypto_processing"
	authservice "github.com/parta4ok/kvs/question/internal/adapter/introspector/auth_service"
	natsbroker "github.com/parta4ok/kvs/question/internal/adapter/message_broker/nats"
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
	storageImpl, sessionStorage, eventStorage := app.initStorage()
	generator := app.initGenerator()
	authClient := app.initAuthServiceClient()
	acc := app.initAccessor()

	service := app.initSessionServiceBase(storageImpl, sessionStorage, generator)
	broker := app.initBroker()

	scheduler := app.initScheduler(eventStorage, broker)

	publicPort := app.initPublicPort(service, authClient, acc)
	privatePort := app.initPrivatePort(service)

	baseApp := baseApplication.NewBuilder(app.cfg).
		WithLogger().
		WithTracer().
		WithPort(publicPort).
		WithPort(privatePort).
		WithPort(scheduler).
		Build()

	if err := baseApp.RunAndAwait(context.Background()); err != nil {
		app.panic(err)
	}
}

func (app *App) initBroker() cases.MessageBroker {
	slog.Info("init broker started")

	subject := app.cfg.GetNatsSubject()

	pub := app.initNatsPub()
	brokerAdapter, err := natsbroker.NewPublisher(pub, subject)
	if err != nil {
		app.panic(err)
	}

	return brokerAdapter
}

func (app *App) initStorage() (cases.Storage, entities.SessionStorage, storage.EventStorage) {
	slog.Info("init storage started")

	var baseStorage cases.Storage
	var sessionStorage entities.SessionStorage
	var eventStorage storage.EventStorage

	storageType := app.cfg.GetServiceStorageType()
	connStr := app.cfg.GetStorageConnStr(storageType)
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

	a, err := accessor.NewRightAccessor()
	if err != nil {
		err := errors.Wrap(err, "new right accessor failure")
		app.panic(err)
	}

	return a
}

func (app *App) initGenerator() entities.IDGenerator {
	slog.Info("init generator started")
	return cryptoprocessing.NewUint64Generator()
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

func (app *App) initSessionServiceBase(
	storageImpl cases.Storage,
	sessionStorage entities.SessionStorage,
	generator entities.IDGenerator,
) cases.SessionService {
	slog.Info("init session_service started")

	respondTime := app.cfg.GetTimeToRespond()

	serv, err := cases.NewSessionServiceBase(storageImpl, sessionStorage, generator,
		cases.WithCustomRespondTime(respondTime))
	if err != nil {
		err := errors.Wrap(err, "NewSessionServiceBase")
		app.panic(err)
	}

	return serv
}

func (app *App) initAuthServiceClient() public.Introspector {
	slog.Info("init auth service client started")

	addr := app.cfg.GetAuthConn()
	if addr == "" {
		err := errors.Wrap(entities.ErrInvalidParam, "get auth address failure")
		app.panic(err)
	}

	client, err := authservice.NewAuthService(addr)
	if err != nil {
		err := errors.Wrap(entities.ErrInvalidParam, "new auth service client failure")
		app.panic(err)
	}

	return client
}

func (app *App) initPublicPort(
	sessionServiceBase cases.SessionService,
	authClient public.Introspector,
	acc public.Accessor,
) *httpport.Port {
	slog.Info("init public port started")

	addr := app.cfg.GetPublicPort()
	timeout := app.cfg.GetPublicTimeout()
	dailyLimit := app.cfg.GetSessionLimit()

	server, err := public.New(
		public.WithCustomDailySessionLimit(dailyLimit),
		public.WithService(sessionServiceBase),
		public.WithIntrospector(authClient),
		public.WithAccessor(acc),
	)
	if err != nil {
		err := errors.Wrap(err, "new public server init failure")
		app.panic(err)
	}

	httpPort, err := httpport.NewPort(
		httpport.Config{Addr: addr, Timeout: timeout},
		httpport.WithType(public.PortType),
		httpport.WithMiddleware(server.IntrospectMiddleware),
		httpport.WithRoutes(server.Routes()...),
	)
	if err != nil {
		err := errors.Wrap(err, "new public port init failure")
		app.panic(err)
	}

	return httpPort
}

func (app *App) initPrivatePort(sessionServiceBase cases.SessionService) *httpport.Port {
	slog.Info("init private port started")

	addr := app.cfg.GetPrivatePort()
	timeout := app.cfg.GetPrivateTimeout()

	server, err := private.New(
		private.WithService(sessionServiceBase),
	)
	if err != nil {
		err := errors.Wrap(err, "new private server init failure")
		app.panic(err)
	}

	httpPort, err := httpport.NewPort(
		httpport.Config{Addr: addr, Timeout: timeout},
		httpport.WithType(private.PortType),
		httpport.WithRoutes(server.Routes()...),
	)
	if err != nil {
		err := errors.Wrap(err, "new private port init failure")
		app.panic(err)
	}

	return httpPort
}

func (app *App) initScheduler(
	eventStorage storage.EventStorage,
	broker cases.MessageBroker,
) *cron.Scheduler {
	scheduler, err := cron.NewScheduler()
	if err != nil {
		app.panic(err)
	}

	if err = scheduler.NewJob(app.cfg.GetPublisherInterval(),
		app.publishEvents, eventStorage, broker); err != nil {
		app.panic(err)
	}
	if err = scheduler.NewJob(app.cfg.GetFlusherInterval(),
		app.flushEvents, eventStorage); err != nil {
		app.panic(err)
	}

	return scheduler
}

func (app *App) panic(err error, args ...any) {
	slog.Error("application panic", append([]any{"error", err.Error()}, args...)...)
	os.Exit(1)
}

func (app *App) publishEvents(eventStorage storage.EventStorage, broker cases.MessageBroker) {
	slog.Info("cron publishEvents start")

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()

	unpublishedEvents, err := eventStorage.GetUnpublishedEvents(ctx)
	if err != nil {
		slog.Warn("get unpublished events", "error", err.Error())
	}

	for _, event := range unpublishedEvents {
		fn := func(ctx context.Context) error {
			if err := broker.Publish(ctx, event); err != nil {
				slog.Warn("publish event", "error", err.Error())
				return err
			}

			return nil
		}

		if err := eventStorage.MarkEventAsPublished(ctx, event.Num(), fn); err != nil {
			slog.Warn("mark event as published", "error", err.Error())
		}
	}

	slog.Info("cron publishEvents finished")
}

func (app *App) flushEvents(eventStorage storage.EventStorage) {
	slog.Info("cron flushEvents start")

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()

	if err := eventStorage.FlushPublishedEvents(ctx); err != nil {
		slog.Warn("flush published events", "error", err.Error())
	}
	slog.Info("cron flushEvents finished")
}
