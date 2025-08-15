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
	slog.Info("Publisher get event for publish in stream", slog.String("subject", subject))

	if _, err := publisher.conn.PublishMsg(&nats.Msg{
		Subject: subject,
		Data:    message,
	}, nats.Context(ctx)); err != nil {
		return errors.Wrapf(ErrInternal, "failed to publish message: %v", err)
	}

	return nil
}
