package nats

import (
	"context"
	"log/slog"

	"github.com/parta4ok/kvs/question/internal/cases"
	"github.com/parta4ok/kvs/question/internal/entities"
	"github.com/parta4ok/kvs/question/internal/entities/event"
	"github.com/parta4ok/kvs/toolkit/pkg/broker/nats/publisher"
	"github.com/parta4ok/kvs/toolkit/pkg/tracer"
	"github.com/pkg/errors"
)

var (
	_ cases.MessageBroker = (*Publisher)(nil)
)

const (
	SessionFinishedEventType = "SessionResultEvent"
)

type Publisher struct {
	pub     *publisher.Publisher
	subject string
}

func NewPublisher(pub *publisher.Publisher, subject string) (*Publisher, error) {
	if subject == "" {
		return nil, errors.Wrapf(entities.ErrInternal, "subject cannot be empty")
	}

	return &Publisher{
		pub:     pub,
		subject: subject,
	}, nil
}

func (p *Publisher) Publish(ctx context.Context, event event.Event) error {
	slog.Info("Publisher: processing event started", "type", event.Type())
	ctx, span, cancel := tracer.Start(ctx, "NATSPublisherEventSpan")
	defer cancel()

	if err := p.pub.Publish(ctx, p.subject, event.Payload()); err != nil {
		if errors.Is(err, publisher.ErrInternal) {
			err = errors.Wrapf(entities.ErrInternal, "publish failure: %v", err)
		}

		if errors.Is(err, publisher.ErrInvalidParam) {
			err = errors.Wrapf(entities.ErrInvalidParam, "publish failure: %v", err)
		}
		slog.Error("publish failure", "error", err.Error())
		span.SetError(err)
		return err
	}

	return nil
}
