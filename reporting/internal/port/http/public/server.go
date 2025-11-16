package public

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/go-chi/chi"
	"github.com/parta4ok/kvs/reporting/internal/entities"
	port "github.com/parta4ok/kvs/reporting/internal/port"
	"github.com/parta4ok/kvs/reporting/pkg/dto"
	"github.com/parta4ok/kvs/toolkit/pkg/accessor"
	"github.com/parta4ok/kvs/toolkit/pkg/tracing"
	"github.com/pkg/errors"
)

const (
	basePath     = "/reporting/v1"
	passedTopics = "/passed-topics"

	rightGetReport = "get_report"
)

type Server struct {
	router       *chi.Mux
	server       *http.Server
	service      port.Service
	introspector Introspector
	accessor     Accessor
	cfg          *ServerCfg
}

type ServerCfg struct {
	Port    string
	Timeout time.Duration
}

type ServerOption func(*Server)

func WithService(srv port.Service) ServerOption {
	return func(s *Server) {
		s.service = srv
	}
}

func WithConfig(cfg *ServerCfg) ServerOption {
	return func(s *Server) {
		s.cfg = cfg
	}
}

func WithIntrospector(introspector Introspector) ServerOption {
	return func(s *Server) {
		s.introspector = introspector
	}
}

func WithAccessor(accessor Accessor) ServerOption {
	return func(s *Server) {
		s.accessor = accessor
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

	if serv.service == nil {
		err := errors.Wrap(entities.ErrInternal, "service not set")
		slog.Error(err.Error())
		return nil, err
	}

	if serv.introspector == nil {
		err := errors.Wrap(entities.ErrInternal, "introspector not set")
		slog.Error(err.Error())
		return nil, err
	}

	if serv.accessor == nil {
		err := errors.Wrap(entities.ErrInternal, "accessor not set")
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
	s.router.Use(
		s.timeoutMiddleware,
		s.introspectMiddleware,
	)

	s.router.Route(basePath, func(r chi.Router) {
		r.Get(fmt.Sprintf("/{mentor_id}%s", passedTopics), s.GetPassedTopics)
	})
}

// Get report about passed topics
//
// @Summary      Get report about passed topics
// @Description  Retrieves a report about passed topics of mentor groups
// @Accept       json
// @Produce      json
// @Security     ApiKeyAuth
// @Param        Authorization header string true "Bearer {token}"
// @Param        mentor_id path string true "Mentor ID"
// @Success      200  {string}  string  "Successfully report generated"
// @Failure      400  {object}  dto.ErrorDTO   "Invalid request parameters"
// @Failure      404  {object}  dto.ErrorDTO   "No topics found"
// @Failure      500  {object}  dto.ErrorDTO   "Internal server error"
// @Router       /{mentor_id}/passed-topics [get]
func (s *Server) GetPassedTopics(resp http.ResponseWriter, req *http.Request) {
	slog.Info("GetPassedTopics started")
	resp.Header().Set("Content-Type", "application/json")
	ctx, span, cancel := tracing.GlobalTracer().Start(req.Context(), "GetPassedTopicsHandlerSpan")
	defer cancel()

	if err := s.checkUserRights(ctx, []string{rightGetReport}); err != nil {
		slog.Error(err.Error())
		span.SetError(err, "checkUserRights")
		s.errProcessing(resp, err)
		return
	}

	mentorID := chi.URLParam(req, "mentor_id")

	if mentorID == "" {
		err := errors.Wrap(entities.ErrInvalidParam, "mentor_id invalid")
		slog.Error(err.Error())
		span.SetError(err, "mentor_id invalid")
		s.errProcessing(resp, err)
		return
	}

	if err := s.service.GetPassedTopicsByGroups(ctx, mentorID); err != nil {
		slog.Error(err.Error())
		span.SetError(err, "GetPassedTopicsByGroups")
		s.errProcessing(resp, err)
		return
	}

	resp.WriteHeader(http.StatusOK)
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
	case errors.Is(err, entities.ErrForbidden) || errors.Is(err, accessor.ErrAssertion):
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
	resp.Write(errDtoData) //nolint:errcheck,gosec //ok
}

func (s *Server) timeoutMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		ctx, cancel := context.WithTimeout(req.Context(), s.cfg.Timeout)
		defer cancel()

		req = req.WithContext(ctx)
		next.ServeHTTP(resp, req)
	})
}

func (s *Server) introspectMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		authHeader := req.Header.Get("Authorization")
		if authHeader == "" {
			err := errors.Wrap(entities.ErrForbidden, "authoriztion header not set")
			slog.Error(err.Error())
			s.errProcessing(resp, err)
			return
		}

		const prefix = "Bearer "
		authorizationData := strings.Split(authHeader, prefix)
		if len(authorizationData) != 2 {
			err := errors.Wrap(entities.ErrForbidden, "authoriztion header invalid")
			slog.Error(err.Error())
			s.errProcessing(resp, err)
			return
		}

		jwt := authorizationData[1]

		claims, err := s.introspector.Introspect(req.Context(), jwt)
		if err != nil {
			err := errors.Wrap(entities.ErrForbidden, "introspection failure")
			slog.Error(err.Error())
			s.errProcessing(resp, err)
			return
		}

		ctx := context.WithValue(req.Context(), accessor.UserClaims, &accessor.Claims{
			Username: claims.Username,
			Issuer:   claims.Issuer,
			Subject:  claims.Subject,
			Audience: claims.Audience,
			Rights:   claims.Rights,
		})
		next.ServeHTTP(resp, req.WithContext(ctx))
	})
}

func (s *Server) checkUserRights(ctx context.Context, requiredRights []string) error {
	hasEnoughRights, err := s.accessor.HasPermission(ctx, requiredRights)
	if err != nil {
		return err
	}

	if !hasEnoughRights {
		err := errors.Wrap(entities.ErrForbidden, "user has not enough rights")
		return err
	}

	return nil
}
