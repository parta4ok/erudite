package main

import (
	"context"
	"flag"
	"log"
	"log/slog"
	"os"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/parta4ok/kvs/toolkit/nats/migrations"
)

type Config struct {
	NatsURL       string
	MigrationMode string
	Timeout       time.Duration
}

func main() {
	config := parseFlags()
	setupLogging()

	nc, js, err := connectToNATS(config.NatsURL, config.Timeout)
	if err != nil {
		log.Fatalf("Failed to connect to NATS: %v", err)
	}
	defer nc.Close()

	slog.Info("Connected to NATS successfully", "url", config.NatsURL)

	runner := migrations.NewMigrationRunner(js)
	loadMigrations(runner)

	ctx, cancel := context.WithTimeout(context.Background(), config.Timeout)
	defer cancel()

	switch config.MigrationMode {
	case "up":
		slog.Info("Running migrations UP")
		if err := runner.RunMigrations(ctx); err != nil {
			slog.Error("Failed to run migrations", "error", err)
			return
		}
		slog.Info("Migrations completed successfully")

	case "down":
		slog.Info("Running migrations DOWN (rollback)")
		if err := runner.RollbackMigrations(ctx); err != nil {
			slog.Error("Failed to rollback migrations", "error", err)
			return
		}
		slog.Info("Migrations rolled back successfully")

	case "status":
		slog.Info("Checking migration status")
		if err := checkMigrationStatus(ctx, js); err != nil {
			slog.Error("Failed to check migration status", "error", err)
			return
		}

	default:
		slog.Error("Unknown migration mode", "mode", config.MigrationMode)
		return
	}

	slog.Info("NATS migration tool completed successfully")
}

func parseFlags() Config {
	config := Config{}

	flag.StringVar(&config.NatsURL, "nats-url", getEnv("NATS_URL", "nats://localhost:4222"),
		"NATS server URL")
	flag.StringVar(&config.MigrationMode, "mode", getEnv("MIGRATION_MODE", "up"),
		"Migration mode: up, down, status")
	flag.DurationVar(&config.Timeout, "timeout", 30*time.Second, "Operation timeout")

	flag.Parse()

	return config
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func setupLogging() {
	opts := &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}

	handler := slog.NewTextHandler(os.Stdout, opts)
	logger := slog.New(handler)
	slog.SetDefault(logger)
}

func connectToNATS(url string, timeout time.Duration) (*nats.Conn, nats.JetStreamContext, error) {
	opts := []nats.Option{
		nats.Name("NATS Migration Tool"),
		nats.Timeout(timeout),
	}

	nc, err := nats.Connect(url, opts...)
	if err != nil {
		return nil, nil, err
	}

	js, err := nc.JetStream()
	if err != nil {
		nc.Close()
		return nil, nil, err
	}

	return nc, js, nil
}

func loadMigrations(runner *migrations.MigrationRunner) {
	slog.Info("Loading NATS migrations...")

	// Stream name constant to ensure consistency
	const streamName = "session_stream"

	sessionStream := migrations.NewStreamMigration(
		"1754841234_session_stream",
		"session stream",
		streamName,
		[]string{"sessions.*"},
	)
	sessionStream.MaxAge = 7 * 24 * time.Hour
	sessionStream.Storage = nats.FileStorage
	runner.AddMigration(sessionStream)

	sessionConsumer := migrations.NewConsumerMigration(
		"1754841235_session_consumer",
		"session consumer",
		streamName, // Using the same stream name as above
		"session-consumer",
	)
	sessionConsumer.FilterSubject = "sessions.*"
	sessionConsumer.DeliverPolicy = nats.DeliverAllPolicy
	sessionConsumer.AckPolicy = nats.AckExplicitPolicy
	runner.AddMigration(sessionConsumer)

	slog.Info("NATS migrations loaded successfully", "count", 2)
}

func checkMigrationStatus(_ context.Context, js nats.JetStreamContext) error {
	slog.Info("Checking NATS migration status...")

	streams := js.StreamNames()
	streamList := make([]string, 0)

	for streamName := range streams {
		streamList = append(streamList, streamName)

		info, err := js.StreamInfo(streamName)
		if err != nil {
			slog.Error("Failed to get stream info", "stream", streamName, "error", err)
			continue
		}

		slog.Info("Stream status",
			"name", streamName,
			"subjects", info.Config.Subjects,
			"messages", info.State.Msgs)

		consumers := js.ConsumerNames(streamName)
		for consumerName := range consumers {
			slog.Info("Consumer found", "stream", streamName, "name", consumerName)
		}
	}

	slog.Info("Migration status check completed", "streams", len(streamList))
	return nil
}
