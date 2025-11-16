package publisher

import (
	"context"
	"log/slog"

	"github.com/nats-io/nats.go"
	"github.com/pkg/errors"
)

var (
	ErrInvalidParam = errors.New("invalid param")
	ErrInternal     = errors.New("internal error")
)

type Publisher struct {
	conn nats.JetStreamContext
}

func NewPublisher(natsUrl string) (*Publisher, error) {
	if natsUrl == "" {
		return nil, errors.Wrap(ErrInvalidParam, "nats url is empty")
	}

	conn, err := nats.Connect(natsUrl)
	if err != nil {
		return nil, errors.Wrapf(ErrInternal, "connection err: %v", err)
	}

	js, err := conn.JetStream()
	if err != nil {
		return nil, errors.Wrapf(ErrInternal, "jetstream creating failure: %v", err)
	}

	return &Publisher{
		conn: js,
	}, nil
}

func (publisher *Publisher) Publish(ctx context.Context, subject string, message []byte) error {
	slog.Info("Publisher get event for publish in stream",
		slog.String("subject", subject),
		slog.Int("message_size", len(message)))

	select {
	case <-ctx.Done():
		slog.Error("Context already cancelled before publish")
		return errors.Wrapf(ErrInternal, "context cancelled: %v", ctx.Err())
	default:
	}

	slog.Info("About to publish message via JetStream")
	ack, err := publisher.conn.PublishMsg(&nats.Msg{
		Subject: subject,
		Data:    message,
	}, nats.Context(ctx))

	if err != nil {
		slog.Error("JetStream publish failed", slog.String("error", err.Error()))
		return errors.Wrapf(ErrInternal, "failed to publish message: %v", err)
	}

	slog.Info("Message published successfully",
		slog.String("stream", ack.Stream),
		slog.Uint64("sequence", ack.Sequence))
	return nil
}
