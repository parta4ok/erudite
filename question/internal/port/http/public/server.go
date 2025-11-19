package public

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi"
	"github.com/parta4ok/kvs/question/internal/entities"
	"github.com/parta4ok/kvs/question/pkg/dto"
	"github.com/parta4ok/kvs/toolkit/pkg/accessor"
	"github.com/parta4ok/kvs/toolkit/pkg/tracing"
	"github.com/pkg/errors"
)

const (
	serverType = "question public"

	basePath                 = "/kvs/v1"
	topicsPath               = "/topics"
	startSessionPath         = "/start_session"
	completeSessionPath      = "/complete_session"
	allCompletedSessionsPath = "/completed_sessions"

	rightViewTopicList         = "view_topic_list"
	rightStartSession          = "start_session"
	rightInfinitySessionsStart = "inifinity_session_start"
	rightCompleteSession       = "complete_session"
	rightViewCompletedSessions = "view_completed_sessions"

	defaultDailySessionLimit = 1
)

type Server struct {
	router       *chi.Mux
	server       *http.Server
	service      Service
	introspector Introspector
	accessor     Accessor
	cfg          *ServerCfg
	sessionLimit int
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

func WithCustomDailySessionLimit(limit int) ServerOption {
	return func(s *Server) {
		s.sessionLimit = limit
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

	if serv.sessionLimit == 0 {
		serv.sessionLimit = defaultDailySessionLimit
	}

	return serv, nil
}

func (s *Server) Start(ctx context.Context) error {
	s.registerRoutes()

	s.server = &http.Server{
		Addr:              s.cfg.Port,
		Handler:           s.router,
		ReadHeaderTimeout: s.cfg.Timeout,
		WriteTimeout:      s.cfg.Timeout,
		IdleTimeout:       s.cfg.Timeout,
	}

	go func() {
		if err := s.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("listen and serve", "error", err)
		}
	}()

	<-ctx.Done()

	return nil
}

func (s *Server) Stop(ctx context.Context) error {
	slog.Info("server will be stopping")

	slog.Info("starting server shutdown process")
	start := time.Now()

	if err := s.server.Shutdown(ctx); err != nil {
		slog.Error("server shutdown", "error", err, "duration", time.Since(start))
		return err
	}

	slog.Info("server stop gracefully", "duration", time.Since(start))

	return nil
}

func (s *Server) Type() string {
	return serverType
}

func (s *Server) registerRoutes() {
	s.router.Use(
		s.timeoutMiddleware,
		s.introspectMiddleware,
	)

	s.router.Get(basePath+topicsPath, s.GetTopics)
	s.router.Get(basePath+"/{user_id}"+allCompletedSessionsPath, s.GetAllCompletedUserSessions)

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
// @Security     ApiKeyAuth
// @Param        Authorization header string true "Bearer {token}"
// @Success      200  {object}  dto.TopicsDTO  "Successfully retrieved list of topics"
// @Failure      400  {object}  dto.ErrorDTO   "Invalid request parameters"
// @Failure      404  {object}  dto.ErrorDTO   "No topics found"
// @Failure      500  {object}  dto.ErrorDTO   "Internal server error"
// @Router       /topics [get]
func (s *Server) GetTopics(resp http.ResponseWriter, req *http.Request) {
	slog.Info("GetTopics started")
	resp.Header().Set("Content-Type", "application/json")
	ctx, span, cancel := tracing.GlobalTracer().Start(req.Context(), "GetTopicsHandlerSpan")
	defer cancel()

	if err := s.checkUserRights(ctx, []string{rightViewTopicList}); err != nil {
		slog.Error(err.Error())
		span.SetError(err, "checkUserRights")
		s.errProcessing(resp, err)
		return
	}

	topics, err := s.service.ShowTopics(ctx)
	if err != nil {
		slog.Error(err.Error())
		span.SetError(err, "ShowTopics")
		s.errProcessing(resp, err)
		return
	}

	topicsDTO := &dto.TopicsDTO{Topics: topics}

	data, err := json.Marshal(topicsDTO)
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
}

// StartSession creates a new testing session for user with selected topics
//
// @Summary      Create new session
// @Description  Starts a new testing session with questions from selected topics
// @Accept       json
// @Produce      json
// @Security     ApiKeyAuth
// @Param        Authorization header string true "Bearer {token}"
// @Param        user_id path int true "User ID"
// @Param        request body dto.TopicsDTO true "Selected topics"
// @Success      201 {object} dto.SessionDTO "Successfully created session"
// @Failure      400 {object} dto.ErrorDTO "Invalid parameters"
// @Failure      404 {object} dto.ErrorDTO "Topics not found"
// @Failure      500 {object} dto.ErrorDTO "Internal server error"
// @Router       /{user_id}/start_session [post]
//
//nolint:funlen //ok
func (s *Server) StartSession(resp http.ResponseWriter, req *http.Request) {
	slog.Info("StartSession started")
	ctx, span, cancel := tracing.GlobalTracer().Start(req.Context(), "StartSessionHandlerSpan")
	defer cancel()

	if err := s.checkUserRights(ctx, []string{rightStartSession}); err != nil {
		slog.Error(err.Error())
		span.SetError(err, "checkUserRights")
		s.errProcessing(resp, err)
		return
	}

	var limit = s.sessionLimit
	if err := s.checkUserRights(req.Context(), []string{rightInfinitySessionsStart}); err == nil {
		limit = 999_999
	}

	resp.Header().Set("Content-Type", "application/json")

	userID := chi.URLParam(req, "user_id")

	if userID == "" {
		err := errors.Wrap(entities.ErrInvalidParam, "userID invalid")
		slog.Error(err.Error())
		span.SetError(err, "userID invalid")
		s.errProcessing(resp, err)
		return
	}

	var topicsDTO dto.TopicsDTO
	if err := json.NewDecoder(req.Body).Decode(&topicsDTO); err != nil {
		err := errors.Wrapf(entities.ErrInvalidParam, "decode req body to topicsDTO failure: %v", err)
		slog.Error(err.Error())
		span.SetError(err, "decode request body")
		s.errProcessing(resp, err)
		return
	}

	sessionID, questions, err := s.service.CreateSession(ctx, userID, topicsDTO.Topics, limit)
	if err != nil {
		err := errors.Wrap(err, "CreateSession failure")
		slog.Error(err.Error())
		span.SetError(err, "CreateSession")
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
		span.SetError(err, "marshal")
		s.errProcessing(resp, err)
		return
	}

	resp.WriteHeader(http.StatusCreated)
	if _, err = resp.Write(data); err != nil {
		err := errors.Wrapf(entities.ErrInternal, "write data to response failure: %v", err)
		slog.Error(err.Error())
		span.SetError(err, "write response")
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
// @Security     ApiKeyAuth
// @Param        Authorization header string true "Bearer {token}"
// @Param        user_id path int true "User ID"
// @Param        session_id path int true "Session ID"
// @Param        request body dto.UserAnswersListDTO true "User answers"
// @Success      200 {object} dto.SessionResultDTO "Successfully completed session"
// @Failure      400 {object} dto.ErrorDTO "Invalid parameters"
// @Failure      404 {object} dto.ErrorDTO "Session not found"
// @Failure      500 {object} dto.ErrorDTO "Internal server error"
// @Router       /{user_id}/{session_id}/complete_session [post]
//
//nolint:funlen //ok
func (s *Server) CompleteSession(resp http.ResponseWriter, req *http.Request) {
	slog.Info("CompleteSession started")
	ctx, span, cancel := tracing.GlobalTracer().Start(req.Context(), "CompleteSessionHandlerSpan")
	defer cancel()

	if err := s.checkUserRights(ctx, []string{rightCompleteSession}); err != nil {
		slog.Error(err.Error())
		span.SetError(err, "checkUserRights")
		s.errProcessing(resp, err)
		return
	}

	resp.Header().Set("Content-Type", "application/json")

	sessionID := chi.URLParam(req, "session_id")

	if sessionID == "" {
		err := errors.Wrap(entities.ErrInvalidParam, "sessionID invalid")
		slog.Error(err.Error())
		span.SetError(err, "sessionID invalid")
		s.errProcessing(resp, err)
		return
	}

	var userAnswersListDTO dto.UserAnswersListDTO
	if err := json.NewDecoder(req.Body).Decode(&userAnswersListDTO); err != nil {
		err := errors.Wrapf(entities.ErrInvalidParam,
			"decode request body to userAnswersListDTO failure: %v", err)
		slog.Error(err.Error())
		span.SetError(err, "decode request body")
		s.errProcessing(resp, err)
		return
	}

	userAnswers := make([]*entities.UserAnswer, 0, len(userAnswersListDTO.AnswersList))
	for _, answerDTO := range userAnswersListDTO.AnswersList {
		userAnswer, err := entities.NewUserAnswer(answerDTO.QuestionID, answerDTO.Answers)
		if err != nil {
			err := errors.Wrapf(entities.ErrInvalidParam, "create user answer failure: %v", err)
			slog.Error(err.Error())
			span.SetError(err, "create user answer")
			s.errProcessing(resp, err)
			return
		}
		userAnswers = append(userAnswers, userAnswer)
	}

	sessionResult, err := s.service.CompleteSession(ctx, sessionID, userAnswers)
	if err != nil {
		err := errors.Wrap(err, "CompleteSession failure")
		slog.Error(err.Error())
		span.SetError(err, "CompleteSession")
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

	slog.Info("CompleteSession completed successfully")
}

// GetAllCompletedUserSessions returns detailed information about all completed sessions for a user.
//
// @Summary      Get all completed sessions for a user
// @Description  Returns detailed list of all completed sessions for the user with the specified id.
// @Accept       json
// @Produce      json
// @Security     ApiKeyAuth
// @Param        Authorization header string true "Bearer {token}"
// @Param        user_id path string true "User ID"
// @Success      200 {object} dto.CompletedSessionsResponseListDTO "List of completed sessions"
// @Failure      400 {object} dto.ErrorDTO "Invalid user_id"
// @Failure      404 {object} dto.ErrorDTO "No completed sessions found"
// @Failure      500 {object} dto.ErrorDTO "Internal server error"
// @Router       /{user_id}/completed_sessions [get]
//
// //nolint:funlen //ok
func (s *Server) GetAllCompletedUserSessions(resp http.ResponseWriter, req *http.Request) {
	slog.Info("GetAllCompletedUserSessions started")
	ctx, span, cancel := tracing.GlobalTracer().Start(req.Context(),
		"GetAllCompletedUserSessionsHandlerSpan")
	defer cancel()

	if err := s.checkUserRights(ctx, []string{rightViewCompletedSessions}); err != nil {
		slog.Error(err.Error())
		span.SetError(err, "checkUserRights")
		s.errProcessing(resp, err)
		return
	}

	resp.Header().Set("Content-Type", "application/json")

	userID := chi.URLParam(req, "user_id")

	if userID == "" {
		err := errors.Wrap(entities.ErrInvalidParam, "userID invalid")
		slog.Error(err.Error())
		span.SetError(err, "userID invalid")
		s.errProcessing(resp, err)
		return
	}

	sessions, err := s.service.GetAllCompletedUserSessions(ctx, userID)
	if err != nil {
		err := errors.Wrap(err, "GetAllCompletedUserSessions failure")
		slog.Error(err.Error())
		span.SetError(err, "GetAllCompletedUserSessions")
		s.errProcessing(resp, err)
		return
	}

	sessionsListDTO := dto.CompletedSessionsResponseListDTO{
		CompletedSessions: make([]dto.CompletedSessionResponseDTO, 0, len(sessions)),
	}
	for _, session := range sessions {
		sessionInfo, err := s.extractDataFromCompleteSession(*session)
		if err != nil {
			err = errors.Wrap(err, "extractDataFromCompleteSession failure")
			slog.Error(err.Error())
			span.SetError(err, "extractDataFromCompleteSession")
			s.errProcessing(resp, err)
			return
		}

		sessionsListDTO.CompletedSessions = append(sessionsListDTO.CompletedSessions, sessionInfo)
	}

	data, err := json.Marshal(sessionsListDTO)
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

func (s *Server) extractDataFromCompleteSession(session entities.Session) (
	dto.CompletedSessionResponseDTO, error) {
	var completeSessionDTO dto.CompletedSessionResponseDTO

	startedAt, err := session.GetStartedAt()
	if err != nil {
		return completeSessionDTO, err
	}

	topics := session.GetTopics()

	answers, err := session.GetUserAnswers()
	if err != nil {
		return completeSessionDTO, err
	}

	answersList := dto.UserAnswersListDTO{
		AnswersList: make([]dto.UserAnswerDTO, 0, len(answers)),
	}

	questions, err := session.GetQuestions()
	if err != nil {
		return completeSessionDTO, err
	}

	questionsMap := make(map[string]entities.Question, len(questions))
	for _, question := range questions {
		questionsMap[question.ID()] = question
	}

	for _, answer := range answers {
		answersList.AnswersList = append(answersList.AnswersList, dto.UserAnswerDTO{
			QuestionID:      answer.GetQuestionID(),
			QuestionSubject: questionsMap[answer.GetQuestionID()].Subject(),
			Answers:         answer.GetSelections(),
		})
	}

	isExpired, err := session.IsExpired()
	if err != nil {
		return completeSessionDTO, err
	}

	result, err := session.GetSessionResult()
	if err != nil {
		return completeSessionDTO, err
	}

	resultDTO := dto.SessionResultDTO{
		IsSuccess: result.IsSuccess,
		Grade:     result.Grade,
	}

	completeSessionDTO.StartedAt = startedAt
	completeSessionDTO.Topics = topics
	completeSessionDTO.UserAnswers = answersList
	completeSessionDTO.IsExpired = isExpired
	completeSessionDTO.SessionResult = resultDTO

	return completeSessionDTO, nil
}
