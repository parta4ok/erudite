package nats

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"
	"time"

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
	subject        string
	ctx            context.Context
	cancel         context.CancelFunc
	wg             *sync.WaitGroup
}

func NewNatsConsumer(conn string, subject string, messageService port.MessageService,
) (*NatsConsumer, error) {
	if conn == "" {
		return nil, errors.Wrap(entities.ErrInternal, "nats connection is nil")
	}

	if subject == "" {
		return nil, errors.Wrap(entities.ErrInternal, "subject cannot be empty")
	}

	if messageService == nil {
		return nil, errors.Wrap(entities.ErrInternal, "message service is nil")
	}

	nc, err := nats.Connect(conn)
	if err != nil {
		return nil, errors.Wrapf(entities.ErrInternal, "connection err: %v", err)
	}

	js, err := nc.JetStream()
	if err != nil {
		return nil, errors.Wrapf(entities.ErrInternal, "failed to get jetstream context: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &NatsConsumer{
		js:             js,
		messageService: messageService,
		subject:        subject,
		ctx:            ctx,
		cancel:         cancel,
		wg:             &sync.WaitGroup{},
	}, nil
}

func (c *NatsConsumer) Start() error {
	slog.Info("Starting NATS consumer for session events")

	sub, err := c.js.Subscribe(c.subject, c.handleMessage, nats.Durable("session-consumer"))
	if err != nil {
		err := errors.Wrap(err, "failed to subscribe to sessions stream")
		slog.Error(err.Error())
		return err
	}

	c.subscription = sub
	slog.Info("NATS consumer started successfully", slog.String("subject", c.subject),
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

	event, err := c.parseMessage(msg)
	if err != nil {
		c.handleParseError(msg, err)
		return
	}

	sessionResult, err := c.createSessionResult(event)
	if err != nil {
		c.handleValidationError(msg, event, err)
		return
	}

	c.processSessionResult(msg, sessionResult)
}

func (c *NatsConsumer) parseMessage(msg *nats.Msg) (*dto.SessionFinishedEvent, error) {
	var eventDTO dto.EventDTO
	if err := json.Unmarshal(msg.Data, &eventDTO); err != nil {
		return nil, errors.Wrapf(entities.ErrInternal, "failed to unmarshal session event: %v", err)
	}

	if eventDTO.EventType == "" {
		return nil, errors.Wrap(entities.ErrInvalidParam, "empty event type")
	}

	event := &dto.SessionFinishedEvent{
		UserID:     eventDTO.Payload.UserID,
		Topics:     eventDTO.Payload.Topics,
		Questions:  eventDTO.Payload.Questions,
		UserAnswer: eventDTO.Payload.UserAnswers,
		IsExpire:   eventDTO.Payload.IsExpire,
		IsSuccess:  eventDTO.Payload.IsSuccess,
		Resume:     eventDTO.Payload.Grade,
	}

	return event, nil
}

func (c *NatsConsumer) createSessionResult(event *dto.SessionFinishedEvent,
) (*entities.SessionResult, error) {
	return entities.NewSessionResult(
		event.UserID,
		event.Topics,
		event.Questions,
		event.UserAnswer,
		event.IsExpire,
		event.IsSuccess,
		event.Resume,
	)
}

func (c *NatsConsumer) handleParseError(msg *nats.Msg, err error) {
	slog.Error(err.Error(), slog.String("subject", msg.Subject))

	slog.Warn("Acknowledging malformed message to prevent retry",
		slog.String("subject", msg.Subject))

	if ackErr := msg.Ack(); ackErr != nil {
		slog.Error("Failed to acknowledge malformed message", "error", ackErr)
	}
}

func (c *NatsConsumer) handleValidationError(msg *nats.Msg, event *dto.SessionFinishedEvent,
	err error) {
	wrappedErr := errors.Wrap(err, "failed to create session result entity")
	slog.Error(wrappedErr.Error(), slog.String("user_id", event.UserID))

	strategy := c.getErrorHandlingStrategy(err)
	strategy.handle(msg, event.UserID)
}

type errorHandlingStrategy interface {
	handle(msg *nats.Msg, userID string)
}

type validationErrorStrategy struct{}

func (s validationErrorStrategy) handle(msg *nats.Msg, userID string) {
	slog.Warn("Acknowledging message with validation error to prevent retry",
		slog.String("user_id", userID))

	if ackErr := msg.Ack(); ackErr != nil {
		slog.Error("Failed to acknowledge message with validation error",
			"error", ackErr, slog.String("user_id", userID))
	}
}

type temporaryErrorStrategy struct{}

func (s temporaryErrorStrategy) handle(msg *nats.Msg, userID string) {
	if nakErr := msg.Nak(); nakErr != nil {
		err := errors.Wrapf(entities.ErrInternal, "failed to nak message: %v", nakErr)
		slog.Error(err.Error(), slog.String("user_id", userID))
	}
}

func (c *NatsConsumer) getErrorHandlingStrategy(err error) errorHandlingStrategy {
	if errors.Is(err, entities.ErrInvalidParam) {
		return validationErrorStrategy{}
	}
	return temporaryErrorStrategy{}
}

func (c *NatsConsumer) processSessionResult(msg *nats.Msg, sessionResult *entities.SessionResult) {
	if err := msg.Ack(); err != nil {
		err := errors.Wrapf(entities.ErrInternal, "failed to ack message: %v", err)
		slog.Error(err.Error(), slog.String("user_id", sessionResult.GetUserID()))
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := c.messageService.SendMessage(ctx, sessionResult); err != nil {
		err := errors.Wrap(err, "failed to send notification")
		slog.Error(err.Error(), slog.String("user_id", sessionResult.GetUserID()))
		return
	}

	slog.Info("Successfully processed session event",
		"user_id", sessionResult.GetUserID(), "subject", msg.Subject)
}
