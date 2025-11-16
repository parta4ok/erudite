package nats

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/parta4ok/kvs/notificationhub/internal/entities"
	port "github.com/parta4ok/kvs/notificationhub/internal/port"
	natsDTO "github.com/parta4ok/kvs/toolkit/pkg/broker/nats"
	"github.com/pkg/errors"
)

const (
	SessionResultEvent = "SessionResultEvent"
)

type NatsConsumer struct {
	js           nats.JetStreamContext
	nc           *nats.Conn
	service      port.MessageService
	subscription *nats.Subscription
	subject      string
	ctx          context.Context
	cancel       context.CancelFunc
	wg           *sync.WaitGroup
}

func NewNatsConsumer(conn string,
	subject string,
	service port.MessageService,
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

func (c *NatsConsumer) Start() error {
	slog.Info("Starting NATS consumer for report events")

	sub, err := c.js.PullSubscribe(c.subject, "report-consumer",
		nats.Bind("report_stream", "report-consumer"))
	if err != nil {
		err := errors.Wrapf(err, "failed to subscribe to %s stream", c.subject)
		slog.Error(err.Error())
		return err
	}

	c.subscription = sub
	slog.Info("NATS consumer started successfully", slog.String("subject", c.subject),
		slog.String("consumer", "report-consumer"))

	c.wg.Add(1)
	go c.processMessages()

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
	c.nc.Close()

	c.wg.Wait()

	slog.Info("NATS consumer stopped successfully")
	return nil
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
	slog.Info("processEvent", slog.String("subject", msg.Subject))

	var reportEventDTO natsDTO.ReportEventDTO
	if err := json.Unmarshal(msg.Data, &reportEventDTO); err != nil {
		if ackErr := msg.Ack(); ackErr != nil {
			slog.Error("Failed to acknowledge malformed message", "error", ackErr)
		}
		return errors.Wrapf(entities.ErrInternal, "failed to unmarshal session event: %v", err)
	}

	if ackErr := msg.Ack(); ackErr != nil {
		slog.Error("Failed to acknowledge malformed message", "error", ackErr)
	}

	recipient, err := entities.NewUser(
		reportEventDTO.Recipient.ID,
		reportEventDTO.Recipient.Contacts,
	)
	if err != nil {
		slog.Error("new user", "error", err)
	}

	event, err := entities.NewNormalEvent(
		reportEventDTO.Kind,
		reportEventDTO.Format,
		reportEventDTO.Payload,
		recipient,
	)
	if err != nil {
		slog.Error("new normal event", "error", err)
	}

	if err := c.service.SendMessage(c.ctx, event); err != nil {
		slog.Error("send message", "error", err)
	}

	return nil
}
