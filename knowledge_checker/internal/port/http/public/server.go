package public

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/go-chi/chi"
	"github.com/parta4ok/kvs/knowledge_checker/internal/entities"
	"github.com/parta4ok/kvs/knowledge_checker/pkg/dto"
	"github.com/pkg/errors"
)

const (
	basePath            = "/kvs/v1"
	topicsPath          = "/topics"
	startSessionPath    = "/start_session"
	completeSessionPath = "/complete_session"
)

type Server struct {
	router  *chi.Mux
	server  *http.Server
	service Service
	cfg     *ServerCfg
}

type ServerCfg struct {
	Port string
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
		err := errors.Wrap(entities.ErrInternal, "servive not set")
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
		Addr:    s.cfg.Port,
		Handler: s.router,
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
	s.router.Get(basePath+topicsPath, s.GetTopics)

	s.router.Route(basePath, func(r chi.Router) {
		r.Post("/{user_id}"+startSessionPath, s.StartSession)
		r.Post("/{user_id}/{session_id}"+completeSessionPath, s.CompleteSession)
	})
}

// Get lists of all existing topics
//
// @Summary      Get all topics
// @Description  Retrieves a list of all available topics in the system
// @Accept       json
// @Produce      json
// @Success      200  {object}  dto.TopicsDTO  "Successfully retrieved list of topics"
// @Failure      400  {object}  dto.ErrorDTO   "Invalid request parameters"
// @Failure      404  {object}  dto.ErrorDTO   "No topics found"
// @Failure      500  {object}  dto.ErrorDTO   "Internal server error"
// @Router       /topics [get]
func (s *Server) GetTopics(resp http.ResponseWriter, req *http.Request) {
	slog.Info("GetTopics started")
	resp.Header().Set("Content-Type", "application/json")

	topics, err := s.service.ShowTopics(req.Context())
	if err != nil {
		slog.Error(err.Error())
		s.errProcessing(resp, err)
		return
	}

	topicsDTO := &dto.TopicsDTO{Topics: topics}

	data, err := json.Marshal(topicsDTO)
	if err != nil {
		err := errors.Wrapf(entities.ErrInternal, "marshal failure: %v", err)
		slog.Error(err.Error())
		s.errProcessing(resp, err)
		return
	}

	resp.WriteHeader(http.StatusOK)
	if _, err = resp.Write(data); err != nil {
		err := errors.Wrapf(entities.ErrInternal, "write data to response failure: %v", err)
		slog.Error(err.Error())
		s.errProcessing(resp, err)
		return
	}
}

// StartSession creates a new testing session for user with selected topics
//
// @Summary      Create new session
// @Description  Starts a new testing session with questions from selected topics
// @Accept       json
// @Produce      json
// @Param        user_id path int true "User ID"
// @Param        request body dto.TopicsDTO true "Selected topics"
// @Success      201 {object} dto.SessionDTO "Successfully created session"
// @Failure      400 {object} dto.ErrorDTO "Invalid parameters"
// @Failure      404 {object} dto.ErrorDTO "Topics not found"
// @Failure      500 {object} dto.ErrorDTO "Internal server error"
// @Router       /{user_id}/start_session [post]
func (s *Server) StartSession(resp http.ResponseWriter, req *http.Request) {
	slog.Info("StartSession started")

	resp.Header().Set("Content-Type", "application/json")

	userID := chi.URLParam(req, "user_id")

	uid, err := strconv.ParseUint(userID, 10, 64)
	if err != nil {
		err := errors.Wrapf(entities.ErrInvalidParam, "userID invalid: %v", err)
		slog.Error(err.Error())
		s.errProcessing(resp, err)
		return
	}

	var topicsDTO dto.TopicsDTO
	if err := json.NewDecoder(req.Body).Decode(&topicsDTO); err != nil {
		err := errors.Wrapf(entities.ErrInvalidParam, "decode req body to topicsDTO failure: %v", err)
		slog.Error(err.Error())
		s.errProcessing(resp, err)
		return
	}

	sessionID, questions, err := s.service.CreateSession(req.Context(), uint64(uid),
		topicsDTO.Topics)
	if err != nil {
		err := errors.Wrap(err, "CreateSession failure")
		slog.Error(err.Error())
		s.errProcessing(resp, err)
		return
	}

	questionsDTO := make([]dto.QuestionDTO, 0, len(questions))
	for _, question := range questions {
		questionsDTO = append(questionsDTO, dto.QuestionDTO{
			ID:           question.ID(),
			QuestionType: question.Type().String(),
			Topic:        question.Topic(),
			Subject:      question.Subject(),
			Variants:     question.Variants(),
		})
	}

	sessionDTO := dto.SessionDTO{
		SessionID: sessionID,
		Topics:    topicsDTO.Topics,
		Questions: questionsDTO,
	}

	data, err := json.Marshal(sessionDTO)
	if err != nil {
		err := errors.Wrapf(entities.ErrInternal, "marshal failure: %v", err)
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

// CompleteSession completes a testing session with user answers
//
// @Summary      Complete session
// @Description  Completes a testing session by submitting user answers and returns session result
// @Accept       json
// @Produce      json
// @Param        user_id path int true "User ID"
// @Param        session_id path int true "Session ID"
// @Param        request body dto.UserAnswersListDTO true "User answers"
// @Success      200 {object} dto.SessionResultDTO "Successfully completed session"
// @Failure      400 {object} dto.ErrorDTO "Invalid parameters"
// @Failure      404 {object} dto.ErrorDTO "Session not found"
// @Failure      500 {object} dto.ErrorDTO "Internal server error"
// @Router       /{user_id}/{session_id}/complete_session [post]
func (s *Server) CompleteSession(resp http.ResponseWriter, req *http.Request) {
	slog.Info("CompleteSession started")

	resp.Header().Set("Content-Type", "application/json")

	sessionID := chi.URLParam(req, "session_id")

	sid, err := strconv.ParseUint(sessionID, 10, 64)
	if err != nil {
		err := errors.Wrapf(entities.ErrInvalidParam, "sessionID invalid: %v", err)
		slog.Error(err.Error())
		s.errProcessing(resp, err)
		return
	}

	var userAnswersListDTO dto.UserAnswersListDTO
	if err := json.NewDecoder(req.Body).Decode(&userAnswersListDTO); err != nil {
		err := errors.Wrapf(entities.ErrInvalidParam,
			"decode request body to userAnswersListDTO failure: %v", err)
		slog.Error(err.Error())
		s.errProcessing(resp, err)
		return
	}

	userAnswers := make([]*entities.UserAnswer, 0, len(userAnswersListDTO.AnswersList))
	for _, answerDTO := range userAnswersListDTO.AnswersList {
		userAnswer, err := entities.NewUserAnswer(answerDTO.QuestionID, answerDTO.Answers)
		if err != nil {
			err := errors.Wrapf(entities.ErrInvalidParam, "create user answer failure: %v", err)
			slog.Error(err.Error())
			s.errProcessing(resp, err)
			return
		}
		userAnswers = append(userAnswers, userAnswer)
	}

	sessionResult, err := s.service.CompleteSession(req.Context(), uint64(sid), userAnswers)
	if err != nil {
		err := errors.Wrap(err, "CompleteSession failure")
		slog.Error(err.Error())
		s.errProcessing(resp, err)
		return
	}

	resultDTO := dto.SessionResultDTO{
		IsSuccess: sessionResult.IsSuccess,
		Grade:     sessionResult.Grade,
	}

	data, err := json.Marshal(resultDTO)
	if err != nil {
		err := errors.Wrapf(entities.ErrInternal, "marshal failure: %v", err)
		slog.Error(err.Error())
		s.errProcessing(resp, err)
		return
	}

	resp.WriteHeader(http.StatusOK)
	if _, err = resp.Write(data); err != nil {
		err := errors.Wrapf(entities.ErrInternal, "write data to response failure: %v", err)
		slog.Error(err.Error())
		s.errProcessing(resp, err)
		return
	}

	slog.Info("CompleteSession completed successfully")
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
