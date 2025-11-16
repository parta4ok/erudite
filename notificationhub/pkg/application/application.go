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

	"github.com/parta4ok/kvs/notificationhub/internal/adapter/config"
	"github.com/parta4ok/kvs/notificationhub/internal/adapter/notifier/mail/base"
	"github.com/parta4ok/kvs/notificationhub/internal/adapter/notifier/telegram"
	"github.com/parta4ok/kvs/notificationhub/internal/cases"
	"github.com/parta4ok/kvs/notificationhub/internal/port"
	natsPort "github.com/parta4ok/kvs/notificationhub/internal/port/nats"
)

type App struct {
	CfgPath      string
	natsConsumer *natsPort.NatsConsumer
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

	mailNotifier := app.initMailNotifier(cfg, nil)
	telegramNotifier := app.initTelegramNotifier(cfg, mailNotifier)

	service := app.initService(cfg, telegramNotifier)

	natsConsumer := app.initNatsConsumer(cfg, service)
	app.natsConsumer = natsConsumer

	if err := natsConsumer.Start(); err != nil {
		slog.Error("Failed to start NATS consumer", "error", err)
		os.Exit(1)
	}

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

func (app *App) initTelegramNotifier(_ *config.Config, nextNotifier cases.Notifier) cases.Notifier {
	var tgNotifier cases.Notifier

	tg, err := telegram.NewTelegramNotifier(nextNotifier, os.Getenv("TG_BOT_TOKEN"))
	if err != nil {
		app.panic(err)
	}

	tgNotifier = tg
	return tgNotifier
}

func (app *App) initMailNotifier(cfg *config.Config, nextNotifier cases.Notifier) cases.Notifier {
	mail := cfg.GetMailSender()
	smtp := cfg.GetMailSenderSMTP()
	port := cfg.GetMailSenderPort()

	var mailNotifier cases.Notifier

	m, err := base.NewMailNotifier(nextNotifier, smtp, mail, port, os.Getenv("EMAIL_PASSWORD"))
	if err != nil {
		app.panic(err)
	}

	mailNotifier = m

	return mailNotifier
}

func (app *App) initService(_ *config.Config, notifier cases.Notifier) port.MessageService {
	slog.Info("init notification service started")

	var service port.MessageService

	srv, err := cases.NewMessageService(notifier)
	if err != nil {
		app.panic(err)
	}

	service = srv

	return service
}

func (app *App) initNatsConsumer(cfg *config.Config, service port.MessageService,
) *natsPort.NatsConsumer {
	conn := cfg.GetNatsURL()
	subject := cfg.GetNatsSubject()

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

func (app *App) startWithGracefulShutdown() {
	ctx, cancel := context.WithCancel(context.Background())
	app.cancel = cancel

	sigOSChan := make(chan os.Signal, 1)
	signal.Notify(sigOSChan, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)

	slog.Info("NotificationHub started successfully")

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

	shutdownTimeout := 5 * time.Second
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer shutdownCancel()

	if app.natsConsumer != nil {
		slog.Info("Stopping NATS consumer...")
		if err := app.natsConsumer.Stop(); err != nil {
			slog.Error("Error stopping NATS consumer", "error", err)
		}
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
