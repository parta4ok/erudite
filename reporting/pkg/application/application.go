package application

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	authservice "github.com/parta4ok/kvs/reporting/internal/adapter/auth_service"
	"github.com/parta4ok/kvs/reporting/internal/adapter/config"
	natsbroker "github.com/parta4ok/kvs/reporting/internal/adapter/message_broker/nats"
	questionservice "github.com/parta4ok/kvs/reporting/internal/adapter/question_service"
	"github.com/parta4ok/kvs/reporting/internal/adapter/representer"
	"github.com/parta4ok/kvs/reporting/internal/cases"
	"github.com/parta4ok/kvs/reporting/internal/entities"
	"github.com/parta4ok/kvs/reporting/internal/port"
	"github.com/parta4ok/kvs/reporting/internal/port/http/public"
	natsconsumer "github.com/parta4ok/kvs/reporting/internal/port/nats"
	"github.com/parta4ok/kvs/toolkit/pkg/accessor"
	baseApplication "github.com/parta4ok/kvs/toolkit/pkg/application"
	"github.com/parta4ok/kvs/toolkit/pkg/broker/nats/publisher"
	httpport "github.com/parta4ok/kvs/toolkit/pkg/port/http"
	natsport "github.com/parta4ok/kvs/toolkit/pkg/port/nats"

	"github.com/pkg/errors"
)

const (
	sessionStream       = "session_stream"
	sessionConsumerName = "session-consumer"
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
	broker := app.initBroker()
	authClient := app.initAuthClient()
	questionClient := app.initQuestionClient()
	repr := app.initRepresenter()
	acc := app.initAccessor()

	service := app.initReportingService(broker, authClient, questionClient, repr)

	publicPort := app.initPublicPort(service, authClient, acc)
	consumerPort := app.initConsumerPort(service)

	baseApp := baseApplication.NewBuilder(app.cfg).
		WithLogger().
		WithTracer().
		WithPort(publicPort).
		WithPort(consumerPort).
		Build()

	if err := baseApp.RunAndAwait(context.Background()); err != nil {
		app.panic(err)
	}
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

func (app *App) initPublicPort(
	service port.Service,
	introspector public.Introspector,
	acc public.Accessor,
) *httpport.Port {
	slog.Info("init public port started")

	addr := app.cfg.GetPublicPort()
	timeout := app.cfg.GetPublicTimeout()

	server, err := public.New(
		public.WithService(service),
		public.WithIntrospector(introspector),
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

func (app *App) initConsumerPort(service port.Service) *natsport.Port {
	slog.Info("init consumer port started")

	consumer, err := natsconsumer.NewConsumer(service)
	if err != nil {
		app.panic(err)
	}

	natsPort, err := natsport.NewPort(
		natsport.Config{
			Conn:    app.cfg.GetNatsURL(),
			Stream:  sessionStream,
			Durable: sessionConsumerName,
			Subject: app.cfg.GetNatsSubject(),
		},
		natsport.WithType("reporting_nats_consumer"),
		natsport.WithHandler(consumer.HandleMessage),
	)
	if err != nil {
		app.panic(err)
	}

	return natsPort
}

func (app *App) initBroker() cases.MessageBroker {
	slog.Info("init broker publisher started")

	pub, err := publisher.NewPublisher(app.cfg.GetNatsURL())
	if err != nil {
		app.panic(err)
	}

	b, err := natsbroker.NewPublisher(pub)
	if err != nil {
		app.panic(err)
	}

	return b
}

func (app *App) initAuthClient() *authservice.AuthService {
	slog.Info("init auth client started")

	authAdapter, err := authservice.NewAuthService(app.cfg.GetAuthConn())
	if err != nil {
		app.panic(err)
	}

	return authAdapter
}

func (app *App) initQuestionClient() cases.QuestionClient {
	slog.Info("init question client started")

	client, err := questionservice.New(app.cfg.GetQuestionConn())
	if err != nil {
		app.panic(err)
	}

	return client
}

func (app *App) initRepresenter() entities.Representer {
	slog.Info("init representer started")

	repr, err := representer.NewRepresenter()
	if err != nil {
		app.panic(err)
	}

	return repr
}

func (app *App) initReportingService(
	broker cases.MessageBroker,
	authClient cases.AuthClient,
	questionClient cases.QuestionClient,
	repr entities.Representer,
) port.Service {
	serv, err := cases.NewReportingService(
		broker,
		repr,
		app.cfg.GetRepresenterFormat(),
		authClient,
		questionClient,
		app.cfg.GetWorkersLimit(),
		app.cfg.GetQueueimit(),
		app.cfg.GetAsyncTimeout(),
	)
	if err != nil {
		app.panic(err)
	}

	return serv
}

func (app *App) panic(err error, args ...any) {
	slog.Error(err.Error(), args...)
	os.Exit(1)
}
