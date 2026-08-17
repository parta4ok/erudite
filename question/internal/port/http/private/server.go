package private

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/parta4ok/kvs/question/internal/entities"
	"github.com/parta4ok/kvs/question/pkg/dto"
	httpport "github.com/parta4ok/kvs/toolkit/pkg/port/http"
	"github.com/parta4ok/kvs/toolkit/pkg/tracer"
	"github.com/pkg/errors"
)

const (
	PortType = "http_private_question"

	basePath                = "/kvs/v1"
	getPassedStudentsTopics = "/passed_topics"
)

type Server struct {
	service Service
}

type ServerOption func(*Server)

func WithService(srv Service) ServerOption {
	return func(s *Server) {
		s.service = srv
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

	return serv, nil
}

func (s *Server) Routes() []httpport.Route {
	return []httpport.Route{
		{
			Method:  http.MethodPost,
			Pattern: basePath + getPassedStudentsTopics,
			Handler: s.GetPassedStudentsTopics,
		},
	}
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
	ctx, span, cancel := tracer.Start(req.Context(),
		"GetPassedStudentsTopicsHandlerSpan")
	defer cancel()

	var studentsDTO dto.StudentsIDsDTO
	if err := json.NewDecoder(req.Body).Decode(&studentsDTO); err != nil {
		err := errors.Wrapf(entities.ErrInvalidParam, "decode req body to studentsDTO failure: %v", err)
		slog.Error("decode req body to studentsDTO failure", "error", err.Error())
		span.SetError(err)
		s.errProcessing(resp, err)
		return
	}

	passedTopicsForStudent, err := s.service.GetPassedStudentsTopics(ctx, studentsDTO.Students)
	if err != nil {
		err := errors.Wrap(err, "GetPassedStudentsTopics")
		slog.Error("GetPassedStudentsTopics", "error", err.Error())
		span.SetError(err)
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
		slog.Error("marshal failure", "error", err.Error())
		span.SetError(err)
		s.errProcessing(resp, err)
		return
	}

	resp.WriteHeader(http.StatusOK)
	if _, err = resp.Write(data); err != nil {
		err := errors.Wrapf(entities.ErrInternal, "write data to response failure: %v", err)
		slog.Error("write data to response failure", "error", err.Error())
		span.SetError(err)
		s.errProcessing(resp, err)
		return
	}

	slog.Info("GetPassedStudentsTopics completed successfully")
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
