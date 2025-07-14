package public

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi"
	"github.com/parta4ok/kvs/auth/internal/entities"
	"github.com/parta4ok/kvs/auth/internal/port"
	"github.com/parta4ok/kvs/auth/pkg/dto"
	"github.com/pkg/errors"
)

const (
	basePath   = "/auth/v1"
	signinPath = "/signin"
)

type Server struct {
	router  *chi.Mux
	server  *http.Server
	factory port.CommandFactory
	cfg     *ServerCfg
}

type ServerCfg struct {
	Port    string
	Timeout time.Duration
}

type ServerOption func(*Server)

func WithFactory(factory port.CommandFactory) ServerOption {
	return func(s *Server) {
		s.factory = factory
	}
}

func WithConfig(cfg *ServerCfg) ServerOption {
	return func(s *Server) {
		s.cfg = cfg
	}
}

func (s *Server) setOption(opts ...ServerOption) {
	for _, opt := range opts {
		opt(s)
	}
}

func New(opts ...ServerOption) (*Server, error) {
	r := chi.NewMux()

	serv := &Server{
		router: r,
	}

	serv.setOption(opts...)

	if serv.factory == nil {
		err := errors.Wrap(entities.ErrInternal, "factory not set")
		slog.Error(err.Error())
		return nil, err
	}

	if serv.cfg == nil {
		err := errors.Wrap(entities.ErrInvalidParam, "config not set")
		slog.Error(err.Error())
		return nil, err
	}

	if serv.cfg.Port == "" {
		err := errors.Wrap(entities.ErrInternal, "port not set")
		slog.Error(err.Error())
		return nil, err
	}

	return serv, nil
}

func (s *Server) Start() {
	slog.Info("public port started")
	s.registerRoutes()

	s.server = &http.Server{
		Addr:              s.cfg.Port,
		Handler:           s.router,
		ReadHeaderTimeout: s.cfg.Timeout,
		WriteTimeout:      s.cfg.Timeout,
		IdleTimeout:       s.cfg.Timeout,
	}

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := s.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error(err.Error())
		}
	}()

	<-done

	s.Stop()
}

func (s *Server) Stop() {
	slog.Info("server will be stopping")

	ctx, cancelFn := context.WithTimeout(context.Background(), time.Second*2)
	defer cancelFn()

	if err := s.server.Shutdown(ctx); err != nil {
		slog.Error(errors.Wrapf(entities.ErrInternal, "shutdown err: %v", err).Error())
	}

	slog.Info("server stop gracefully")
}

func (s *Server) registerRoutes() {
	s.router.Use(s.timeoutMiddleware)

	s.router.Post(basePath+signinPath, s.Signin)
}

// Sign in user
//
// @Summary      Sign in
// @Description  Authenticates user with provided credentials and returns JWT token
// @Accept       json
// @Produce      json
// @Param        request body dto.SigninRequestDTO true "User credentials"
// @Success      201  {object}  dto.SigninResponseDTO "JWT created"
// @Failure      400  {object}  dto.ErrorDTO "Invalid request parameters"
// @Failure      401  {object}  dto.ErrorDTO "Unauthorized"
// @Failure      500  {object}  dto.ErrorDTO "Internal server error"
// @Router       /auth/v1/signin [post]
func (s *Server) Signin(resp http.ResponseWriter, req *http.Request) {
	slog.Info("Signin started")
	resp.Header().Set("Content-Type", "application/json")

	var requestDTO dto.SigninRequestDTO
	if err := json.NewDecoder(req.Body).Decode(&requestDTO); err != nil {
		err := errors.Wrapf(entities.ErrInvalidParam,
			"decode req body to signinRequestDTO failure: %v", err)
		slog.Error(err.Error())
		s.errProcessing(resp, err)
		return
	}

	command, err := s.factory.NewSignInCommand(req.Context(), requestDTO.Login,
		requestDTO.Password)
	if err != nil {
		err := errors.Wrap(err, "signin command creating failure")
		slog.Error(err.Error())
		s.errProcessing(resp, err)
		return
	}

	res, err := command.Exec()
	if err != nil {
		err := errors.Wrap(err, "signin command executing failure")
		slog.Error(err.Error())
		s.errProcessing(resp, err)
		return
	}

	if res == nil {
		err := errors.Wrap(entities.ErrInternal, "signin command executing completed with nil result")
		slog.Error(err.Error())
		s.errProcessing(resp, err)
		return
	}

	if !res.Success {
		err := errors.Wrap(entities.ErrInternal, "signin command executing completed with bad status")
		slog.Error(err.Error())
		s.errProcessing(resp, err)
		return
	}

	topicsDTO := &dto.SigninResponseDTO{Token: res.Message}

	data, err := json.Marshal(topicsDTO)
	if err != nil {
		err := errors.Wrapf(entities.ErrInternal, "marshal token failure: %v", err)
		slog.Error(err.Error())
		s.errProcessing(resp, err)
		return
	}

	resp.WriteHeader(http.StatusCreated)
	if _, err = resp.Write(data); err != nil {
		err := errors.Wrapf(entities.ErrInternal, "write data to response failure: %v", err)
		slog.Error(err.Error())
		s.errProcessing(resp, err)
		return
	}
}

func (s *Server) errProcessing(resp http.ResponseWriter, err error) {
	stausCode := http.StatusInternalServerError
	errDTO := dto.ErrorDTO{
		StatusCode: stausCode,
		ErrMsg:     err.Error(),
	}

	switch {
	case errors.Is(err, entities.ErrInvalidParam):
		errDTO.StatusCode = http.StatusBadRequest
	case errors.Is(err, entities.ErrForbidden):
		errDTO.StatusCode = http.StatusForbidden
	case errors.Is(err, entities.ErrNotFound):
		errDTO.StatusCode = http.StatusNotFound
	}

	errDtoData, err := json.Marshal(&errDTO)
	if err != nil {
		err := errors.Wrapf(entities.ErrInternal, "marshal failure: %v", err)
		slog.Error(err.Error())
		http.Error(resp, err.Error(), http.StatusInternalServerError)
		return
	}

	resp.WriteHeader(errDTO.StatusCode)
	resp.Write(errDtoData) //nolint:errcheck //ok
}

func (s *Server) timeoutMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		ctx, cancel := context.WithTimeout(req.Context(), s.cfg.Timeout)
		defer cancel()

		req = req.WithContext(ctx)
		next.ServeHTTP(resp, req)
	})
}
