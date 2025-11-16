package nats

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/parta4ok/kvs/question/internal/cases"
	"github.com/parta4ok/kvs/question/internal/entities"
	natsDTO "github.com/parta4ok/kvs/toolkit/pkg/broker/nats"
	"github.com/parta4ok/kvs/toolkit/pkg/broker/nats/publisher"
	"github.com/parta4ok/kvs/toolkit/pkg/tracing"
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

func (p *Publisher) SessionFinishedEvent(ctx context.Context,
	sessionResult *entities.SessionResult) error {
	slog.Info("Publisher: SessionFinishedEvent started")
	ctx, span, cancel := tracing.GlobalTracer().Start(ctx, "NATSPublisherSessionFinishedEventSpan")
	defer cancel()
	data := natsDTO.SessionResultDTO{
		UserID:      sessionResult.UserID,
		Topics:      sessionResult.Topics,
		Questions:   sessionResult.Questions,
		UserAnswers: sessionResult.UserAnswers,
		IsExpire:    sessionResult.IsExpire,
		IsSuccess:   sessionResult.IsSuccess,
		Grade:       sessionResult.Grade,
	}

	bytes, err := json.Marshal(data)
	if err != nil {
		err = errors.Wrapf(entities.ErrInternal, "failed to marshal session result: %v", err)
		slog.Error(err.Error())
		span.SetError(err, "failed to marshal session result")
		return err
	}

	event := natsDTO.EventDTO{
		EventType: SessionFinishedEventType,
		Payload:   bytes,
	}

	message, err := json.Marshal(event)
	if err != nil {
		err = errors.Wrapf(entities.ErrInternal, "failed to marshal event: %v", err)
		slog.Error(err.Error())
		span.SetError(err, "failed to marshal event")
		return err
	}

	if err = p.pub.Publish(ctx, p.subject, message); err != nil {
		if errors.Is(err, publisher.ErrInternal) {
			err = errors.Wrapf(entities.ErrInternal, "publish failure: %v", err)
		}

		if errors.Is(err, publisher.ErrInvalidParam) {
			err = errors.Wrapf(entities.ErrInvalidParam, "publish failure: %v", err)
		}
		slog.Error(err.Error())
		span.SetError(err, "publish failure")
		return err
	}

	return nil
}
