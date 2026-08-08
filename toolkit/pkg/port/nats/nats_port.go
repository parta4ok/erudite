package nats

import (
	"context"
	"log/slog"
	"time"

	"github.com/nats-io/nats.go"
	toolkitport "github.com/parta4ok/kvs/toolkit/pkg/port"
	"github.com/pkg/errors"
)

var _ toolkitport.BasePort = (*Port)(nil)

var ErrInvalidParam = errors.New("invalid param")

const (
	defaultFetchBatch   = 10
	defaultFetchTimeout = time.Second
)

type Handler func(msg *nats.Msg)

type Config struct {
	Conn         string
	Stream       string
	Durable      string
	Subject      string
	FetchBatch   int
	FetchTimeout time.Duration
}

type Port struct {
	cfg      Config
	portType string
	handler  Handler

	nc           *nats.Conn
	subscription *nats.Subscription
}

type Option func(*Port)

func WithHandler(h Handler) Option {
	return func(p *Port) { p.handler = h }
}

func WithType(portType string) Option {
	return func(p *Port) { p.portType = portType }
}

func NewPort(cfg Config, opts ...Option) (*Port, error) {
	if cfg.Conn == "" {
		return nil, errors.WithMessage(ErrInvalidParam, "conn not set")
	}

	if cfg.Subject == "" {
		return nil, errors.WithMessage(ErrInvalidParam, "subject not set")
	}

	if cfg.Stream == "" {
		return nil, errors.WithMessage(ErrInvalidParam, "stream not set")
	}

	if cfg.Durable == "" {
		return nil, errors.WithMessage(ErrInvalidParam, "durable consumer name not set")
	}

	if cfg.FetchBatch == 0 {
		cfg.FetchBatch = defaultFetchBatch
	}

	if cfg.FetchTimeout == 0 {
		cfg.FetchTimeout = defaultFetchTimeout
	}

	p := &Port{cfg: cfg, portType: "nats_consumer"}

	for _, opt := range opts {
		opt(p)
	}

	if p.handler == nil {
		return nil, errors.WithMessage(ErrInvalidParam, "handler not set")
	}

	return p, nil
}

func (p *Port) Start(ctx context.Context) error {
	nc, err := nats.Connect(p.cfg.Conn)
	if err != nil {
		return errors.Wrapf(err, "nats connect failure")
	}
	p.nc = nc

	js, err := nc.JetStream()
	if err != nil {
		return errors.Wrapf(err, "jetstream context failure")
	}

	sub, err := js.PullSubscribe(p.cfg.Subject, p.cfg.Durable,
		nats.Bind(p.cfg.Stream, p.cfg.Durable))
	if err != nil {
		return errors.Wrapf(err, "pull subscribe failure")
	}
	p.subscription = sub

	slog.Info("nats consumer started", "type", p.portType, "subject", p.cfg.Subject)

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		msgs, err := p.subscription.Fetch(p.cfg.FetchBatch, nats.MaxWait(p.cfg.FetchTimeout))
		if err != nil {
			if errors.Is(err, nats.ErrTimeout) {
				continue
			}
			if ctx.Err() != nil {
				return nil //nolint:nilerr //ok - ctx already cancelled, this is a graceful stop
			}
			slog.Error("nats fetch failure", "type", p.portType, "error", err)
			continue
		}

		for _, msg := range msgs {
			p.handler(msg)
		}
	}
}

func (p *Port) Stop(_ context.Context) error {
	if p.subscription != nil {
		if err := p.subscription.Unsubscribe(); err != nil {
			slog.Error("nats unsubscribe failure", "type", p.portType, "error", err)
		}
	}

	if p.nc != nil {
		p.nc.Close()
	}

	return nil
}

func (p *Port) Type() string {
	return p.portType
}
