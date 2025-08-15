package nats

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"

	"github.com/nats-io/nats.go"
	"github.com/parta4ok/kvs/notificationhub/internal/entities"
	"github.com/parta4ok/kvs/notificationhub/internal/port"
	"github.com/parta4ok/kvs/notificationhub/pkg/dto"
	"github.com/pkg/errors"
)

type NatsConsumer struct {
	js             nats.JetStreamContext
	messageService port.MessageService
	subscription   *nats.Subscription
	ctx            context.Context
	cancel         context.CancelFunc
	wg             *sync.WaitGroup
}

func NewNatsConsumer(nc *nats.Conn, messageService port.MessageService) (*NatsConsumer, error) {
	if nc == nil {
		return nil, errors.Wrap(entities.ErrInternal, "nats connection is nil")
	}

	if messageService == nil {
		return nil, errors.Wrap(entities.ErrInternal, "message service is nil")
	}

	js, err := nc.JetStream()
	if err != nil {
		return nil, errors.Wrapf(entities.ErrInternal, "failed to get jetstream context: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &NatsConsumer{
		js:             js,
		messageService: messageService,
		ctx:            ctx,
		cancel:         cancel,
		wg:             &sync.WaitGroup{},
	}, nil
}

func (c *NatsConsumer) Start() error {
	slog.Info("Starting NATS consumer for session events")

	sub, err := c.js.Subscribe("sessions.*", c.handleMessage, nats.Durable("session-consumer"))
	if err != nil {
		err := errors.Wrap(err, "failed to subscribe to sessions stream")
		slog.Error(err.Error())
		return err
	}

	c.subscription = sub
	slog.Info("NATS consumer started successfully", slog.String("subject", "sessions.*"),
		slog.String("consumer", "session-consumer"))
	return nil
}

func (c *NatsConsumer) Stop() error {
	slog.Info("Stopping NATS consumer")

	c.cancel()

	if c.subscription != nil {
		if err := c.subscription.Unsubscribe(); err != nil {
			err := errors.Wrapf(entities.ErrInternal, "failed to unsubscribe from NATS: %v", err)
			slog.Error(err.Error())
			return err
		}
	}

	c.wg.Wait()

	slog.Info("NATS consumer stopped successfully")
	return nil
}

func (c *NatsConsumer) handleMessage(msg *nats.Msg) {
	slog.Info("Received message", slog.String("subject", msg.Subject))

	c.wg.Add(1)
	defer c.wg.Done()

	select {
	case <-c.ctx.Done():
		slog.Info("Skipping message processing due to shutdown")
		return
	default:
	}

	var event dto.SessionFinishedEvent
	if err := json.Unmarshal(msg.Data, &event); err != nil {
		err := errors.Wrapf(entities.ErrInternal, "failed to unmarshal session event: %v", err)
		slog.Error(err.Error(), slog.String("subject", msg.Subject))
		if err := msg.Nak(); err != nil {
			err := errors.Wrapf(entities.ErrInternal, "failed to nak message: %v", err)
			slog.Error(err.Error())
		}

		return
	}

	sessionResult, err := entities.NewSessionResult(
		event.UserID,
		event.Topics,
		event.Questions,
		event.UserAnswer,
		event.IsExpire,
		event.IsSuccess,
		event.Resume,
	)
	if err != nil {
		err := errors.Wrap(err, "failed to create session result entity")
		slog.Error(err.Error(), slog.String("user_id", event.UserID))
		if err := msg.Nak(); err != nil {
			err := errors.Wrapf(entities.ErrInternal, "failed to nak message: %v", err)
			slog.Error(err.Error(), slog.String("user_id", event.UserID))
		}
		return
	}

	if err := c.messageService.SendMessage(sessionResult); err != nil {
		err := errors.Wrap(err, "failed to send notification")
		slog.Error(err.Error(), slog.String("user_id", event.UserID))
		if err := msg.Nak(); err != nil {
			err := errors.Wrapf(entities.ErrInternal, "failed to nak message: %v", err)
			slog.Error(err.Error(), slog.String("user_id", event.UserID))
		}
		return
	}

	if err := msg.Ack(); err != nil {
		err := errors.Wrapf(entities.ErrInternal, "failed to ack message: %v", err)
		slog.Error(err.Error(), slog.String("user_id", event.UserID))
		return
	}

	slog.Info("Successfully processed session event", "user_id", event.UserID, "subject", msg.Subject)
}
