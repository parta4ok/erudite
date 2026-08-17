package nats

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/parta4ok/kvs/reporting/internal/entities"
	port "github.com/parta4ok/kvs/reporting/internal/port"
	natsDTO "github.com/parta4ok/kvs/toolkit/pkg/broker/nats"
	"github.com/parta4ok/kvs/toolkit/pkg/tracer"
	"github.com/pkg/errors"
)

const (
	SessionResultEvent = "SessionResultEvent"
)

type Consumer struct {
	service port.Service
}

func NewConsumer(service port.Service) (*Consumer, error) {
	if service == nil {
		return nil, errors.Wrap(entities.ErrInternal, "service not set")
	}

	return &Consumer{service: service}, nil
}

func (c *Consumer) HandleMessage(msg *nats.Msg) {
	slog.Info("Received message", slog.String("subject", msg.Subject))

	if err := c.processEvent(msg); err != nil {
		slog.Error("Failed to process event", "error", err.Error())
	}
}

func (c *Consumer) processEvent(msg *nats.Msg) error {
	var eventDTO natsDTO.EventDTO
	if err := json.Unmarshal(msg.Data, &eventDTO); err != nil {
		if ackErr := msg.Ack(); ackErr != nil {
			slog.Error("Failed to acknowledge malformed message", "error", ackErr.Error())
		}
		return errors.Wrapf(entities.ErrInternal, "failed to unmarshal session event: %v", err)
	}

	switch eventDTO.EventType {
	case SessionResultEvent:
		var sessionResultDTO natsDTO.SessionResultDTO
		if err := json.Unmarshal(eventDTO.Payload, &sessionResultDTO); err != nil {
			if ackErr := msg.Ack(); ackErr != nil {
				slog.Error("Failed to acknowledge malformed message", "error", ackErr.Error())
			}
			return errors.Wrap(entities.ErrInternal, "failed to cast session event")
		}

		ctx := tracer.ExtractNATS(context.Background(), msg)
		ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()

		if err := c.service.DeliverySessionResult(ctx, &entities.SessionResult{
			UserID:     sessionResultDTO.UserID,
			Topics:     sessionResultDTO.Topics,
			Questions:  sessionResultDTO.Questions,
			UserAnswer: sessionResultDTO.UserAnswers,
			IsExpire:   sessionResultDTO.IsExpire,
			IsSuccess:  sessionResultDTO.IsSuccess,
			Resume:     sessionResultDTO.Grade,
		}); err != nil {
			if nakErr := msg.Nak(); nakErr != nil {
				slog.Error("Failed to nak buiseness message", "error", nakErr.Error())
			}
			return errors.Wrap(err, "DeliverySessionResult failure")
		}

		if ackErr := msg.Ack(); ackErr != nil {
			slog.Error("Failed to acknowledge processed message", "error", ackErr.Error())
		}
	default:
		if ackErr := msg.Ack(); ackErr != nil {
			slog.Error("Failed to acknowledge malformed message", "error", ackErr.Error())
		}
	}

	return nil
}
