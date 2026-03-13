package nats

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/parta4ok/kvs/reporting/internal/entities"
	port "github.com/parta4ok/kvs/reporting/internal/port"
	natsDTO "github.com/parta4ok/kvs/toolkit/pkg/broker/nats"
	"github.com/parta4ok/kvs/toolkit/pkg/tracing/middleware"
	"github.com/pkg/errors"
)

const (
	consumerType = "reporting nats consumer"

	SessionResultEvent = "SessionResultEvent"
)

type NatsConsumer struct {
	js           nats.JetStreamContext
	nc           *nats.Conn
	service      port.Service
	subscription *nats.Subscription
	subject      string
	ctx          context.Context
	cancel       context.CancelFunc
	wg           *sync.WaitGroup
}

func NewNatsConsumer(conn string,
	subject string,
	service port.Service,
) (*NatsConsumer, error) {
	if conn == "" {
		return nil, errors.Wrap(entities.ErrInternal, "nats connection is nil")
	}

	if subject == "" {
		return nil, errors.Wrap(entities.ErrInternal, "subject cannot be empty")
	}

	if service == nil {
		return nil, errors.Wrap(entities.ErrInternal, "service not set")
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
		js:      js,
		nc:      nc,
		service: service,
		subject: subject,
		ctx:     ctx,
		cancel:  cancel,
		wg:      &sync.WaitGroup{},
	}, nil
}

func (c *NatsConsumer) Start(_ context.Context) error {
	slog.Info("Starting NATS consumer for session events")

	sub, err := c.js.PullSubscribe(c.subject, "session-consumer",
		nats.Bind("session_stream", "session-consumer"))
	if err != nil {
		err := errors.Wrapf(err, "failed to subscribe to %s stream", c.subject)
		slog.Error(err.Error())
		return err
	}

	c.subscription = sub
	slog.Info("NATS consumer started successfully", slog.String("subject", c.subject),
		slog.String("consumer", "session-consumer"))

	c.wg.Add(1)
	go c.processMessages()

	return nil
}

func (c *NatsConsumer) Stop(ctx context.Context) error {
	slog.Info("stopping nats consumer")

	errChan := make(chan error, 1)
	successChan := make(chan struct{}, 1)

	stopFn := func() {
		c.cancel()

		if c.subscription != nil {
			if err := c.subscription.Unsubscribe(); err != nil {
				err := errors.Wrapf(entities.ErrInternal, "failed to unsubscribe from NATS: %v", err)
				slog.Error(err.Error())
				errChan <- err
				return
			}
		}
		c.nc.Close()

		c.wg.Wait()

		slog.Info("nats consumer stopped successfully")
		close(successChan)
		close(errChan)
	}

	go stopFn()

	select {
	case <-ctx.Done():
		slog.Error("skipping message processing due to shutdown", "error", ctx.Err().Error())
		return errors.Wrap(entities.ErrInternal, "context exceeded, consumer stopped with failure")
	case err := <-errChan:
		slog.Error("consumer stopped func return error", "error", err.Error())
		return errors.Wrap(err, "consumer stopped func return error")
	case <-successChan:
		slog.Info("consumer successfull stop")
		return nil
	}
}

func (c *NatsConsumer) Type() string {
	return consumerType
}

func (c *NatsConsumer) processMessages() {
	defer c.wg.Done()

	for {
		select {
		case <-c.ctx.Done():
			return
		default:
			msgs, err := c.subscription.Fetch(10, nats.MaxWait(1*time.Second))
			if err != nil && !errors.Is(err, nats.ErrTimeout) {
				slog.Error("Failed to fetch messages", "error", err)
				continue
			}

			for _, msg := range msgs {
				c.handleMessage(msg)
			}
		}
	}
}

func (c *NatsConsumer) handleMessage(msg *nats.Msg) {
	slog.Info("Received message", slog.String("subject", msg.Subject))

	select {
	case <-c.ctx.Done():
		slog.Info("Skipping message processing due to shutdown")
		return
	default:
	}

	if err := c.processEvent(msg); err != nil {
		slog.Error("Failed to process event", "error", err)
	}

}

func (c *NatsConsumer) processEvent(msg *nats.Msg) error {
	var eventDTO natsDTO.EventDTO
	if err := json.Unmarshal(msg.Data, &eventDTO); err != nil {
		if ackErr := msg.Ack(); ackErr != nil {
			slog.Error("Failed to acknowledge malformed message", "error", ackErr)
		}
		return errors.Wrapf(entities.ErrInternal, "failed to unmarshal session event: %v", err)
	}

	switch eventDTO.EventType {
	case SessionResultEvent:
		var sessionResultDTO natsDTO.SessionResultDTO
		if err := json.Unmarshal(eventDTO.Payload, &sessionResultDTO); err != nil {
			if ackErr := msg.Ack(); ackErr != nil {
				slog.Error("Failed to acknowledge malformed message", "error", ackErr)
			}
			return errors.Wrap(entities.ErrInternal, "failed to cast session event")
		}

		ctx := middleware.ExtractTraceFromNatsMessage(context.Background(), msg)
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
				slog.Error("Failed to nak buiseness message", "error", nakErr)
			}
			return errors.Wrap(err, "DeliverySessionResult failure")
		}

		if ackErr := msg.Ack(); ackErr != nil {
			slog.Error("Failed to acknowledge processed message", "error", ackErr)
		}
	default:
		if ackErr := msg.Ack(); ackErr != nil {
			slog.Error("Failed to acknowledge malformed message", "error", ackErr)
		}
	}

	return nil
}
