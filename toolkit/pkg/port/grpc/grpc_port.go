package grpc

import (
	"context"
	"net"

	"google.golang.org/grpc"

	toolkitport "github.com/parta4ok/kvs/toolkit/pkg/port"
	"github.com/pkg/errors"
)

var _ toolkitport.BasePort = (*Port)(nil)

var ErrInvalidParam = errors.New("invalid param")

type RegisterFunc func(*grpc.Server)

type Config struct {
	Addr string
}

type Port struct {
	cfg      Config
	portType string

	registerFuncs []RegisterFunc
	serverOpts    []grpc.ServerOption

	server   *grpc.Server
	listener net.Listener
}

type Option func(*Port)

func WithRegister(fn RegisterFunc) Option {
	return func(p *Port) { p.registerFuncs = append(p.registerFuncs, fn) }
}

func WithServerOptions(opts ...grpc.ServerOption) Option {
	return func(p *Port) { p.serverOpts = append(p.serverOpts, opts...) }
}

func WithType(portType string) Option {
	return func(p *Port) { p.portType = portType }
}

func NewPort(cfg Config, opts ...Option) (*Port, error) {
	if cfg.Addr == "" {
		return nil, errors.WithMessage(ErrInvalidParam, "addr not set")
	}

	p := &Port{cfg: cfg, portType: "grpc"}

	for _, opt := range opts {
		opt(p)
	}

	if len(p.registerFuncs) == 0 {
		return nil, errors.WithMessage(ErrInvalidParam, "no service registered")
	}

	return p, nil
}

func (p *Port) Start(_ context.Context) error {
	listener, err := net.Listen("tcp", p.cfg.Addr)
	if err != nil {
		return errors.Wrapf(err, "net listen failure")
	}
	p.listener = listener

	p.server = grpc.NewServer(p.serverOpts...)
	for _, register := range p.registerFuncs {
		register(p.server)
	}

	return p.server.Serve(listener) //nolint:wrapcheck //ok
}

func (p *Port) Stop(_ context.Context) error {
	if p.server != nil {
		p.server.GracefulStop()
	}

	return nil
}

func (p *Port) Type() string {
	return p.portType
}
