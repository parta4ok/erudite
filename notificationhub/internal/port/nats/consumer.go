package nats

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/nats-io/nats.go"
	"github.com/parta4ok/kvs/notificationhub/internal/entities"
	port "github.com/parta4ok/kvs/notificationhub/internal/port"
	natsDTO "github.com/parta4ok/kvs/toolkit/pkg/broker/nats"
	"github.com/parta4ok/kvs/toolkit/pkg/tracer"
	"github.com/pkg/errors"
)

const (
	SessionResultEvent = "SessionResultEvent"
)

type Consumer struct {
	service port.MessageService
}

func NewConsumer(service port.MessageService) (*Consumer, error) {
	if service == nil {
		return nil, errors.Wrap(entities.ErrInternal, "service not set")
	}

	return &Consumer{service: service}, nil
}

// HandleMessage is passed to toolkit/pkg/port/nats.Port as the fetch handler.
func (c *Consumer) HandleMessage(msg *nats.Msg) {
	slog.Info("Received message", slog.String("subject", msg.Subject))

	if err := c.processEvent(msg); err != nil {
		slog.Error("Failed to process event", "error", err)
	}
}

func (c *Consumer) processEvent(msg *nats.Msg) error {
	ctx := tracer.ExtractNATS(context.Background(), msg)

	var reportEventDTO natsDTO.ReportEventDTO
	if err := json.Unmarshal(msg.Data, &reportEventDTO); err != nil {
		if ackErr := msg.Ack(); ackErr != nil {
			slog.Error("Failed to acknowledge malformed message", "error", ackErr)
		}
		return errors.Wrapf(entities.ErrInternal, "failed to unmarshal session event: %v", err)
	}

	recipient, err := entities.NewUser(
		reportEventDTO.Recipient.ID,
		reportEventDTO.Recipient.Contacts,
	)
	if err != nil {
		if ackErr := msg.Ack(); ackErr != nil {
			slog.Error("Failed to acknowledge malformed message", "error", ackErr)
		}
		slog.Error("new user", "error", err)
		return err
	}

	event, err := entities.NewNormalEvent(
		reportEventDTO.Kind,
		reportEventDTO.Format,
		reportEventDTO.Payload,
		recipient,
	)
	if err != nil {
		if ackErr := msg.Ack(); ackErr != nil {
			slog.Error("Failed to acknowledge malformed message", "error", ackErr)
		}
		slog.Error("new normal event", "error", err)
		return err
	}

	if err := c.service.SendMessage(ctx, event); err != nil {
		if nakErr := msg.Nak(); nakErr != nil {
			slog.Error("Failed to nak buiseness message", "error", nakErr)
		}
		slog.Error("send message", "error", err)
		return err
	}

	if ackErr := msg.Ack(); ackErr != nil {
		slog.Error("Failed to acknowledge processed message", "error", ackErr)
	}

	return nil
}
