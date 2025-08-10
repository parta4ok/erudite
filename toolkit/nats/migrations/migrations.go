package migrations

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/pkg/errors"
)

type MigrationType string

const (
	MigrationTypeStream   MigrationType = "stream"
	MigrationTypeConsumer MigrationType = "consumer"
)

type Migration interface {
	GetID() string
	GetName() string
	GetType() MigrationType
	Apply(ctx context.Context, js nats.JetStreamContext) error
	Rollback(ctx context.Context, js nats.JetStreamContext) error
}

type BaseMigration struct {
	ID   string
	Name string
	Type MigrationType
}

func (m *BaseMigration) GetID() string {
	return m.ID
}

func (m *BaseMigration) GetName() string {
	return m.Name
}

func (m *BaseMigration) GetType() MigrationType {
	return m.Type
}

type StreamMigration struct {
	BaseMigration
	StreamConfig nats.StreamConfig
	Subjects     []string
	MaxAge       time.Duration
	Storage      nats.StorageType
}

func NewStreamMigration(id, name string, streamName string, subjects []string) *StreamMigration {
	return &StreamMigration{
		BaseMigration: BaseMigration{
			ID:   id,
			Name: name,
			Type: MigrationTypeStream,
		},
		StreamConfig: nats.StreamConfig{
			Name:     streamName,
			Subjects: subjects,
		},
		Subjects: subjects,
		MaxAge:   24 * time.Hour,
		Storage:  nats.FileStorage,
	}
}

func (m *StreamMigration) Apply(ctx context.Context, js nats.JetStreamContext) error {
	config := m.StreamConfig
	config.Subjects = m.Subjects
	config.MaxAge = m.MaxAge
	config.Storage = m.Storage

	_, err := js.StreamInfo(config.Name)
	if err != nil {
		if errors.Is(err, nats.ErrStreamNotFound) {
			slog.Info("Creating new stream", "name", config.Name, "subjects", config.Subjects)
			_, err = js.AddStream(&config)
			if err != nil {
				return errors.Wrapf(err, "failed to create stream %s", config.Name)
			}
			slog.Info("Stream created successfully", "name", config.Name)
			return nil
		}
		return errors.Wrapf(err, "failed to get stream info for %s", config.Name)
	}

	slog.Info("Updating existing stream", "name", config.Name, "subjects", config.Subjects)
	_, err = js.UpdateStream(&config)
	if err != nil {
		return errors.Wrapf(err, "failed to update stream %s", config.Name)
	}
	slog.Info("Stream updated successfully", "name", config.Name)
	return nil
}

func (m *StreamMigration) Rollback(ctx context.Context, js nats.JetStreamContext) error {
	slog.Info("Deleting stream", "name", m.StreamConfig.Name)
	err := js.DeleteStream(m.StreamConfig.Name)
	if err != nil && !errors.Is(err, nats.ErrStreamNotFound) {
		return errors.Wrapf(err, "failed to delete stream %s", m.StreamConfig.Name)
	}
	slog.Info("Stream deleted successfully", "name", m.StreamConfig.Name)
	return nil
}

type ConsumerMigration struct {
	BaseMigration
	StreamName     string
	ConsumerConfig nats.ConsumerConfig
	DeliverPolicy  nats.DeliverPolicy
	AckPolicy      nats.AckPolicy
	FilterSubject  string
}

func NewConsumerMigration(id, name, streamName, consumerName string) *ConsumerMigration {
	return &ConsumerMigration{
		BaseMigration: BaseMigration{
			ID:   id,
			Name: name,
			Type: MigrationTypeConsumer,
		},
		StreamName: streamName,
		ConsumerConfig: nats.ConsumerConfig{
			Name: consumerName,
		},
		DeliverPolicy: nats.DeliverAllPolicy,
		AckPolicy:     nats.AckExplicitPolicy,
	}
}

func (m *ConsumerMigration) Apply(ctx context.Context, js nats.JetStreamContext) error {
	config := m.ConsumerConfig
	config.DeliverPolicy = m.DeliverPolicy
	config.AckPolicy = m.AckPolicy
	config.FilterSubject = m.FilterSubject

	_, err := js.ConsumerInfo(m.StreamName, config.Name)
	if err != nil {
		if errors.Is(err, nats.ErrConsumerNotFound) {
			slog.Info("Creating new consumer",
				"stream", m.StreamName,
				"consumer", config.Name,
				"deliver_policy", config.DeliverPolicy)
			_, err = js.AddConsumer(m.StreamName, &config)
			if err != nil {
				return errors.Wrapf(err, "failed to create consumer %s for stream %s",
					config.Name, m.StreamName)
			}
			slog.Info("Consumer created successfully",
				"stream", m.StreamName, "consumer", config.Name)
			return nil
		}
		return errors.Wrapf(err, "failed to get consumer info for %s/%s",
			m.StreamName, config.Name)
	}

	slog.Info("Consumer already exists",
		"stream", m.StreamName, "consumer", config.Name)
	return nil
}

func (m *ConsumerMigration) Rollback(ctx context.Context, js nats.JetStreamContext) error {
	slog.Info("Deleting consumer",
		"stream", m.StreamName, "consumer", m.ConsumerConfig.Name)
	err := js.DeleteConsumer(m.StreamName, m.ConsumerConfig.Name)
	if err != nil && !errors.Is(err, nats.ErrConsumerNotFound) {
		return errors.Wrapf(err, "failed to delete consumer %s from stream %s",
			m.ConsumerConfig.Name, m.StreamName)
	}
	slog.Info("Consumer deleted successfully",
		"stream", m.StreamName, "consumer", m.ConsumerConfig.Name)
	return nil
}

type MigrationRunner struct {
	js         nats.JetStreamContext
	migrations []Migration
}

func NewMigrationRunner(js nats.JetStreamContext) *MigrationRunner {
	return &MigrationRunner{
		js:         js,
		migrations: make([]Migration, 0),
	}
}

func (r *MigrationRunner) AddMigration(migration Migration) {
	r.migrations = append(r.migrations, migration)
}

func (r *MigrationRunner) AddMigrations(migrations ...Migration) {
	r.migrations = append(r.migrations, migrations...)
}

//nolint:dupl //ok
func (r *MigrationRunner) RunMigrations(ctx context.Context) error {
	if len(r.migrations) == 0 {
		slog.Info("No migrations to run")
		return nil
	}

	sort.Slice(r.migrations, func(i, j int) bool {
		return r.migrations[i].GetID() < r.migrations[j].GetID()
	})

	slog.Info("Starting NATS migrations", "count", len(r.migrations))

	for _, migration := range r.migrations {
		slog.Info("Applying migration", "id", migration.GetID(), "name", migration.GetName())

		if err := migration.Apply(ctx, r.js); err != nil {
			return errors.Wrapf(err, "failed to apply migration %s", migration.GetID())
		}
	}

	slog.Info("All NATS migrations completed successfully")
	return nil
}

//nolint:dupl //ok
func (r *MigrationRunner) RollbackMigrations(ctx context.Context) error {
	if len(r.migrations) == 0 {
		slog.Info("No migrations to rollback")
		return nil
	}

	sort.Slice(r.migrations, func(i, j int) bool {
		return r.migrations[i].GetID() > r.migrations[j].GetID()
	})

	slog.Info("Starting NATS migrations rollback", "count", len(r.migrations))

	for _, migration := range r.migrations {
		slog.Info("Rolling back migration", "id", migration.GetID(), "name", migration.GetName())

		if err := migration.Rollback(ctx, r.js); err != nil {
			return errors.Wrapf(err, "failed to rollback migration %s", migration.GetID())
		}
	}

	slog.Info("All NATS migrations rolled back successfully")
	return nil
}

func GenerateMigrationID(name string) string {
	timestamp := time.Now().Unix()
	return fmt.Sprintf("%d_%s", timestamp, name)
}

func ParseMigrationID(id string) (int64, string, error) {
	parts := strings.Split(id, "_")
	if len(parts) < 2 {
		return 0, "", errors.New("invalid migration ID format")
	}

	timestamp, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return 0, "", errors.Wrap(err, "invalid timestamp in migration ID")
	}

	name := strings.Join(parts[1:], "_")
	return timestamp, name, nil
}

func ValidateMigrationID(id string) error {
	_, _, err := ParseMigrationID(id)
	return err
}
