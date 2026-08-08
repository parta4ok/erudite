package http

import (
	"context"
	"net/http"
	"time"

	"github.com/go-chi/chi"
	toolkitport "github.com/parta4ok/kvs/toolkit/pkg/port"
	"github.com/pkg/errors"
)

var _ toolkitport.BasePort = (*Port)(nil)

var ErrInvalidParam = errors.New("invalid param")

type Route struct {
	Method  string
	Pattern string
	Handler http.HandlerFunc
}

type Config struct {
	Addr        string
	Timeout     time.Duration
	TLSCertFile string
	TLSKeyFile  string
}

type Port struct {
	cfg         Config
	portType    string
	routes      []Route
	middlewares []func(http.Handler) http.Handler

	router *chi.Mux
	server *http.Server
}

type Option func(*Port)

func WithRoutes(routes ...Route) Option {
	return func(p *Port) { p.routes = append(p.routes, routes...) }
}

func WithMiddleware(mw ...func(http.Handler) http.Handler) Option {
	return func(p *Port) { p.middlewares = append(p.middlewares, mw...) }
}

func WithType(portType string) Option {
	return func(p *Port) { p.portType = portType }
}

func NewPort(cfg Config, opts ...Option) (*Port, error) {
	if cfg.Addr == "" {
		return nil, errors.WithMessagef(ErrInvalidParam, "addr not set")
	}

	p := &Port{
		cfg:      cfg,
		portType: "http",
	}

	for _, opt := range opts {
		opt(p)
	}

	return p, nil
}

func (p *Port) Start(_ context.Context) error {
	p.router = chi.NewMux()
	p.router.Use(p.middlewares...)
	p.router.Use(p.timeoutMiddleware)

	for _, r := range p.routes {
		p.router.MethodFunc(r.Method, r.Pattern, r.Handler)
	}

	p.server = &http.Server{
		Addr:              p.cfg.Addr,
		Handler:           p.router,
		ReadHeaderTimeout: p.cfg.Timeout,
		WriteTimeout:      p.cfg.Timeout,
		IdleTimeout:       p.cfg.Timeout,
	}

	var err error
	if p.cfg.TLSCertFile != "" {
		err = p.server.ListenAndServeTLS(p.cfg.TLSCertFile, p.cfg.TLSKeyFile)
	} else {
		err = p.server.ListenAndServe()
	}

	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}

	return err
}

func (p *Port) Stop(ctx context.Context) error {
	if p.server == nil {
		return nil
	}
	return p.server.Shutdown(ctx)
}

func (p *Port) Type() string {
	return p.portType
}

func (p *Port) timeoutMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		ctx, cancel := context.WithTimeout(req.Context(), p.cfg.Timeout)
		defer cancel()
		next.ServeHTTP(resp, req.WithContext(ctx))
	})
}
