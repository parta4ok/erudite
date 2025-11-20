package application

import (
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/parta4ok/kvs/notificationhub/internal/adapter/config"
	"github.com/parta4ok/kvs/notificationhub/internal/adapter/notifier/mail/base"
	"github.com/parta4ok/kvs/notificationhub/internal/adapter/notifier/telegram"
	"github.com/parta4ok/kvs/notificationhub/internal/cases"
	"github.com/parta4ok/kvs/notificationhub/internal/entities"
	"github.com/parta4ok/kvs/notificationhub/internal/port"
	natsPort "github.com/parta4ok/kvs/notificationhub/internal/port/nats"
	baseApplication "github.com/parta4ok/kvs/toolkit/pkg/application"
	"github.com/parta4ok/kvs/toolkit/pkg/logger"
	"github.com/parta4ok/kvs/toolkit/pkg/tracing"
	projectTracer "github.com/parta4ok/kvs/toolkit/pkg/tracing/jaeger"
	otelTracer "github.com/parta4ok/kvs/toolkit/pkg/tracing/otel"

	"github.com/pkg/errors"
)

type App struct {
	cfg          *config.Config
	natsConsumer *natsPort.NatsConsumer
	tracerPort   *tracing.TracerPort
	tracing      bool
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

	mailNotifier := app.initMailNotifier(nil)
	telegramNotifier := app.initTelegramNotifier(mailNotifier)

	service := app.initService(telegramNotifier)

	natsConsumer := app.initNatsConsumer(service)
	app.natsConsumer = natsConsumer

	baseApp := baseApplication.NewBaseApplication()
	baseApp.SetTimeout(5 * time.Second)
	baseApp.SetPorts([]baseApplication.Port{
		app.natsConsumer,
		app.tracerPort,
	})

	baseApp.StartPortsWithGracefulShutdown()
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

func (app *App) initTelegramNotifier(nextNotifier cases.Notifier) cases.Notifier {
	tg, err := telegram.NewTelegramNotifier(nextNotifier, os.Getenv("TG_BOT_TOKEN"))
	if err != nil {
		app.panic(err)
	}

	return tg
}

func (app *App) initMailNotifier(nextNotifier cases.Notifier) cases.Notifier {
	mail := app.cfg.GetMailSender()
	smtp := app.cfg.GetMailSenderSMTP()
	port := app.cfg.GetMailSenderPort()

	m, err := base.NewMailNotifier(nextNotifier, smtp, mail, port, os.Getenv("EMAIL_PASSWORD"))
	if err != nil {
		app.panic(err)
	}

	return m
}

func (app *App) initService(notifier cases.Notifier) port.MessageService {
	slog.Info("init notification service started")

	srv, err := cases.NewMessageService(notifier)
	if err != nil {
		app.panic(err)
	}

	return srv
}

func (app *App) initNatsConsumer(service port.MessageService) *natsPort.NatsConsumer {
	slog.Info("init nats consumer started")

	conn := app.cfg.GetNatsURL()
	subject := app.cfg.GetNatsSubject()

	consumer, err := natsPort.NewNatsConsumer(conn, subject, service)
	if err != nil {
		app.panic(err)
	}

	return consumer
}

func (app *App) panic(err error, args ...any) {
	slog.Error(err.Error(), args...)
	os.Exit(1)
}
