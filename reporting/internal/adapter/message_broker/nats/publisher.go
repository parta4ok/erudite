package nats

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/parta4ok/kvs/reporting/internal/cases"
	"github.com/parta4ok/kvs/reporting/internal/entities"
	natsDTO "github.com/parta4ok/kvs/toolkit/pkg/broker/nats"
	"github.com/parta4ok/kvs/toolkit/pkg/broker/nats/publisher"
	"github.com/parta4ok/kvs/toolkit/pkg/tracer"

	"github.com/pkg/errors"
)

var (
	_ cases.MessageBroker = (*Publisher)(nil)
)

type Publisher struct {
	pub *publisher.Publisher
}

func NewPublisher(pub *publisher.Publisher) (*Publisher, error) {
	return &Publisher{
		pub: pub,
	}, nil
}

func (p *Publisher) ReportEvent(ctx context.Context, event entities.Event) error {
	ctx, span, cancel := tracer.Start(ctx, "NATSReportEventSpan")
	defer cancel()

	slog.Info("Publisher: SessionFinishedEvent started")

	eventDTO := natsDTO.ReportEventDTO{
		Kind:      event.Kind().String(),
		Format:    event.Format().String(),
		Payload:   event.Payload(),
		Recipient: natsDTO.UserDTO(*event.Recipient()),
	}

	message, err := json.Marshal(eventDTO)
	if err != nil {
		err = errors.Wrapf(entities.ErrInternal, "failed to marshal payload: %v", err)
		slog.Error(err.Error())
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
		slog.Error(err.Error())
		span.SetError(err)
		return err
	}

	return nil
}
