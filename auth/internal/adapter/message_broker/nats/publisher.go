package nats

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/parta4ok/kvs/auth/internal/cases/common"
	"github.com/parta4ok/kvs/auth/internal/entities"
	natsDTO "github.com/parta4ok/kvs/toolkit/pkg/broker/nats"
	"github.com/parta4ok/kvs/toolkit/pkg/broker/nats/publisher"
	"github.com/parta4ok/kvs/toolkit/pkg/tracer"

	"github.com/pkg/errors"
)

var (
	_ common.MessageBroker = (*Publisher)(nil)
)

type Publisher struct {
	pub *publisher.Publisher
}

func NewPublisher(pub *publisher.Publisher) (*Publisher, error) {
	return &Publisher{
		pub: pub,
	}, nil
}

func (p *Publisher) SendEvent(ctx context.Context, event entities.Event) error {
	ctx, span, cancel := tracer.Start(ctx, "SendEventToNatsSpan")
	defer cancel()

	slog.Info("Publisher: DynamicRegistrationEvent started")

	eventDTO := natsDTO.DynamicRegistrationEventDTO{
		Kind:      event.Kind().String(),
		Format:    "text",
		Payload:   event.Payload(),
		Recipient: natsDTO.DynamicUserDTO(*event.Recipient()),
	}

	slog.Info("new event dto", slog.Any("event", eventDTO))

	message, err := json.Marshal(eventDTO)
	if err != nil {
		err = errors.Wrapf(entities.ErrInternal, "failed to marshal payload: %v", err)
		slog.Error("failed to marshal payload", "error", err)
		span.SetError(err)
		return err
	}

	if err = p.pub.Publish(ctx, event.Kind().String(), message); err != nil {
		if errors.Is(err, publisher.ErrInternal) {
			err = errors.Wrapf(entities.ErrInternal, "publish failure: %v", err)
		}

		if errors.Is(err, publisher.ErrInvalidParam) {
			err = errors.Wrapf(entities.ErrInvalidParam, "publish failure: %v", err)
		}
		slog.Error("publish failure", "error", err)
		span.SetError(err)
		return err
	}

	return nil
}
