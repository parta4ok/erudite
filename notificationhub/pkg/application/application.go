package application

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/parta4ok/kvs/notificationhub/internal/adapter/config"
	"github.com/parta4ok/kvs/notificationhub/internal/adapter/notifier/mail/base"
	"github.com/parta4ok/kvs/notificationhub/internal/adapter/notifier/telegram"
	"github.com/parta4ok/kvs/notificationhub/internal/cases"
	"github.com/parta4ok/kvs/notificationhub/internal/port"
	natsconsumer "github.com/parta4ok/kvs/notificationhub/internal/port/nats"
	baseApplication "github.com/parta4ok/kvs/toolkit/pkg/application"
	natsport "github.com/parta4ok/kvs/toolkit/pkg/port/nats"
)

const (
	reportStream       = "report_stream"
	reportConsumerName = "report-consumer"
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
	mailNotifier := app.initMailNotifier(nil)
	telegramNotifier := app.initTelegramNotifier(mailNotifier)

	service := app.initService(telegramNotifier)

	consumerPort := app.initConsumerPort(service)

	baseApp := baseApplication.NewBuilder(app.cfg).
		WithLogger().
		WithTracer().
		WithPort(consumerPort).
		Build()

	if err := baseApp.RunAndAwait(context.Background()); err != nil {
		app.panic(err)
	}
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
	mailPort := app.cfg.GetMailSenderPort()

	m, err := base.NewMailNotifier(nextNotifier, smtp, mail, mailPort, os.Getenv("EMAIL_PASSWORD"))
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

func (app *App) initConsumerPort(service port.MessageService) *natsport.Port {
	slog.Info("init consumer port started")

	consumer, err := natsconsumer.NewConsumer(service)
	if err != nil {
		app.panic(err)
	}

	natsPort, err := natsport.NewPort(
		natsport.Config{
			Conn:    app.cfg.GetNatsURL(),
			Stream:  reportStream,
			Durable: reportConsumerName,
			Subject: app.cfg.GetNatsSubject(),
		},
		natsport.WithType("notificationhub_nats_consumer"),
		natsport.WithHandler(consumer.HandleMessage),
	)
	if err != nil {
		app.panic(err)
	}

	return natsPort
}

func (app *App) panic(err error, args ...any) {
	slog.Error("application panic", append([]any{"error", err.Error()}, args...)...)
	os.Exit(1)
}
