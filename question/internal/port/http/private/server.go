package private

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi"
	"github.com/parta4ok/kvs/question/internal/entities"
	"github.com/parta4ok/kvs/question/pkg/dto"
	"github.com/parta4ok/kvs/toolkit/pkg/tracing"
	"github.com/pkg/errors"
)

const (
	basePath                = "/kvs/v1"
	getPassedStudentsTopics = "/passed_topics"
)

type Server struct {
	router  *chi.Mux
	server  *http.Server
	service Service
	cfg     *ServerCfg
}

type ServerCfg struct {
	Port    string
	Timeout time.Duration
}

type ServerOption func(*Server)

func WithService(srv Service) ServerOption {
	return func(s *Server) {
		s.service = srv
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

	if serv.service == nil {
		err := errors.Wrap(entities.ErrInternal, "service not set")
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
	)

	s.router.Post(basePath+getPassedStudentsTopics, s.GetPassedStudentsTopics)
}

func (s *Server) timeoutMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		ctx, cancel := context.WithTimeout(req.Context(), s.cfg.Timeout)
		defer cancel()

		req = req.WithContext(ctx)
		next.ServeHTTP(resp, req)
	})
}

// GetPassedStudentsTopics returns detailed information about all passed topics for
// specified students.
//
// @Summary      Get passed topics for multiple students
// @Description  Returns a map of student IDs to their completed topics.
// Accepts a list of student IDs in the request body.
// @Accept       json
// @Produce      json
// @Param        students body dto.StudentsIDsDTO true "List of student IDs"
// @Success      200 {object} dto.StudentsTopicsDTO "Map of student IDs to their passed topics"
// @Failure      400 {object} dto.ErrorDTO "Bad request - invalid student IDs format"
// @Failure      500 {object} dto.ErrorDTO "Internal server error"
// @Router       /passed_topics [post]
func (s *Server) GetPassedStudentsTopics(resp http.ResponseWriter, req *http.Request) {
	slog.Info("GetPassedStudentsTopics started")
	resp.Header().Set("Content-Type", "application/json")
	ctx, span, cancel := tracing.GlobalTracer().Start(req.Context(),
		"GetPassedStudentsTopicsHandlerSpan")
	defer cancel()

	var studentsDTO dto.StudentsIDsDTO
	if err := json.NewDecoder(req.Body).Decode(&studentsDTO); err != nil {
		err := errors.Wrapf(entities.ErrInvalidParam, "decode req body to studentsDTO failure: %v", err)
		slog.Error(err.Error())
		span.SetError(err, "decode request body")
		s.errProcessing(resp, err)
		return
	}

	passedTopicsForStudent, err := s.service.GetPassedStudentsTopics(ctx, studentsDTO.Students)
	if err != nil {
		err := errors.Wrap(err, "GetPassedStudentsTopics")
		slog.Error(err.Error())
		span.SetError(err, "GetPassedStudentsTopics")
		s.errProcessing(resp, err)
		return
	}
	var passedTopicsDTO = dto.StudentsTopicsDTO{
		StudentsTopics: make(map[string][]dto.TopicWithIDDTO, len(passedTopicsForStudent)),
	}
	for student, topics := range passedTopicsForStudent {
		for _, topic := range topics {
			if _, ok := passedTopicsDTO.StudentsTopics[student]; !ok {
				passedTopicsDTO.StudentsTopics[student] = make([]dto.TopicWithIDDTO, 0, len(topics))
			}
			topicDTO := dto.TopicWithIDDTO{
				ID:    fmt.Sprintf("%d", topic.ID),
				Title: topic.Title,
			}

			passedTopicsDTO.StudentsTopics[student] = append(
				passedTopicsDTO.StudentsTopics[student], topicDTO)
		}
	}

	data, err := json.Marshal(passedTopicsDTO)
	if err != nil {
		err := errors.Wrapf(entities.ErrInternal, "marshal failure: %v", err)
		slog.Error(err.Error())
		span.SetError(err, "marshal")
		s.errProcessing(resp, err)
		return
	}

	resp.WriteHeader(http.StatusOK)
	if _, err = resp.Write(data); err != nil {
		err := errors.Wrapf(entities.ErrInternal, "write data to response failure: %v", err)
		slog.Error(err.Error())
		span.SetError(err, "write response")
		s.errProcessing(resp, err)
		return
	}

	slog.Info("GetPassedStudentsTopics completed successfully")
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
