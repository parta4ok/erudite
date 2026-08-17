package public

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-chi/chi"
	"github.com/parta4ok/kvs/reporting/internal/entities"
	port "github.com/parta4ok/kvs/reporting/internal/port"
	"github.com/parta4ok/kvs/reporting/pkg/dto"
	"github.com/parta4ok/kvs/toolkit/pkg/accessor"
	httpport "github.com/parta4ok/kvs/toolkit/pkg/port/http"
	"github.com/parta4ok/kvs/toolkit/pkg/tracer"
	"github.com/pkg/errors"
)

const (
	PortType = "http_public_reporting"

	basePath     = "/reporting/v1"
	passedTopics = "/passed-topics"

	rightGetReport = "get_report"
)

type Server struct {
	service      port.Service
	introspector Introspector
	accessor     Accessor
}

type ServerOption func(*Server)

func WithService(srv port.Service) ServerOption {
	return func(s *Server) {
		s.service = srv
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
	serv := &Server{}

	serv.setOption(opts...)

	if serv.service == nil {
		err := errors.Wrap(entities.ErrInternal, "service not set")
		slog.Error("service not set", "error", err.Error())
		return nil, err
	}

	if serv.introspector == nil {
		err := errors.Wrap(entities.ErrInternal, "introspector not set")
		slog.Error("introspector not set", "error", err.Error())
		return nil, err
	}

	if serv.accessor == nil {
		err := errors.Wrap(entities.ErrInternal, "accessor not set")
		slog.Error("accessor not set", "error", err.Error())
		return nil, err
	}

	return serv, nil
}

func (s *Server) Routes() []httpport.Route {
	return []httpport.Route{
		{
			Method:  http.MethodGet,
			Pattern: fmt.Sprintf("%s/{mentor_id}%s", basePath, passedTopics),
			Handler: s.GetPassedTopics,
		},
	}
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
	ctx, span, cancel := tracer.Start(req.Context(), "GetPassedTopicsHandlerSpan")
	defer cancel()

	slog.Info("GetPassedTopics started")
	resp.Header().Set("Content-Type", "application/json")

	if err := s.checkUserRights(ctx, []string{rightGetReport}); err != nil {
		slog.Error("checkUserRights failure", "error", err.Error())
		span.SetError(err)
		s.errProcessing(resp, err)
		return
	}

	mentorID := chi.URLParam(req, "mentor_id")

	if mentorID == "" {
		err := errors.Wrap(entities.ErrInvalidParam, "mentor id invalid")
		slog.Error("mentor id invalid", "error", err.Error())
		span.SetError(err)
		s.errProcessing(resp, err)
		return
	}

	if err := s.service.GetPassedTopicsByGroups(ctx, mentorID); err != nil {
		slog.Error("GetPassedTopicsByGroups failure", "error", err.Error())
		span.SetError(err)
		s.errProcessing(resp, err)
		return
	}

	resp.WriteHeader(http.StatusOK)
}

func (s *Server) errProcessing(resp http.ResponseWriter, err error) {
	statusCode := http.StatusInternalServerError
	errDTO := dto.ErrorDTO{
		StatusCode: statusCode,
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
		slog.Error("marshal failure", "error", err.Error())
		http.Error(resp, err.Error(), http.StatusInternalServerError)
		return
	}

	resp.WriteHeader(errDTO.StatusCode)
	resp.Write(errDtoData) //nolint:errcheck,gosec //ok
}

func (s *Server) IntrospectMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		authHeader := req.Header.Get("Authorization")
		if authHeader == "" {
			err := errors.Wrap(entities.ErrForbidden, "authoriztion header not set")
			slog.Error("authoriztion header not set", "error", err.Error())
			s.errProcessing(resp, err)
			return
		}

		const prefix = "Bearer "
		authorizationData := strings.Split(authHeader, prefix)
		if len(authorizationData) != 2 {
			err := errors.Wrap(entities.ErrForbidden, "authoriztion header invalid")
			slog.Error("authoriztion header invalid", "error", err.Error())
			s.errProcessing(resp, err)
			return
		}

		jwt := authorizationData[1]

		claims, err := s.introspector.Introspect(req.Context(), jwt)
		if err != nil {
			err := errors.Wrap(entities.ErrForbidden, "introspection failure")
			slog.Error("introspection failure", "error", err.Error())
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
