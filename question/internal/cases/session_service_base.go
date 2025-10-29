package cases

import (
	"context"
	"log/slog"
	"time"

	"github.com/pkg/errors"

	"github.com/parta4ok/kvs/question/internal/entities"

	"github.com/parta4ok/kvs/toolkit/pkg/tracing"
)

const (
	defaultTopicDuration = time.Minute * 10
)

type SessionServiceBase struct {
	storage        Storage
	sessionStorage entities.SessionStorage
	generator      entities.IDGenerator
	respondTime    time.Duration
}

func NewSessionServiceBase(storage Storage, sessionStorage entities.SessionStorage,
	generator entities.IDGenerator, opts ...SessionServiceOption) (*SessionServiceBase, error) {
	if storage == nil {
		return nil, errors.Wrapf(entities.ErrInvalidParam, "storage not set")
	}

	if sessionStorage == nil {
		return nil, errors.Wrapf(entities.ErrInvalidParam, "session storage not set")
	}

	if generator == nil {
		return nil, errors.Wrapf(entities.ErrInvalidParam, "generator not set")
	}

	service := &SessionServiceBase{
		storage:        storage,
		sessionStorage: sessionStorage,
		generator:      generator,
		respondTime:    defaultTopicDuration,
	}

	service.setOptions(opts...)

	return service, nil
}

type SessionServiceOption func(*SessionServiceBase)

func WithCustomRespondTime(dur time.Duration) SessionServiceOption {
	return func(srv *SessionServiceBase) {
		srv.respondTime = dur
	}
}

func (srv *SessionServiceBase) setOptions(opts ...SessionServiceOption) {
	for _, opt := range opts {
		opt(srv)
	}
}

func (srv *SessionServiceBase) ShowTopics(ctx context.Context) ([]string, error) {
	slog.Info("ShowTopics started")
	ctx, span, cancel := tracing.GlobalTracer().Start(ctx, "ShowTopicsSpan")
	defer cancel()

	topics, err := srv.storage.GetTopics(ctx)
	if err != nil {
		slog.Error(err.Error())
		span.SetError(err, "GetTopics")
		return nil, errors.Wrap(err, "GetTopics")
	}

	slog.Info("ShowTopics completed")
	return topics, nil
}

func (srv *SessionServiceBase) CreateSession(ctx context.Context, userID string,
	topics []string, dailyLimit int) (string, map[string]entities.Question, error) {
	slog.Info("CreateSession started")
	ctx, span, cancel := tracing.GlobalTracer().Start(ctx, "CreateSessionSpan")
	defer cancel()

	session, err := entities.NewSession(userID, topics, srv.generator, srv.sessionStorage)
	if err != nil {
		slog.Error(err.Error())
		span.SetError(err, "NewSession")
		return "", nil, errors.Wrap(err, "NewSession")
	}

	forbidded, err := session.IsDailySessionLimitReached(ctx, userID, topics, dailyLimit)
	if err != nil {
		slog.Error(err.Error())
		span.SetError(err, "IsDailySessionLimitReached")
		return "", nil, errors.Wrap(err, "IsDailySessionLimitReached")
	}

	if forbidded {
		err := errors.Wrap(entities.ErrForbidden, "creating new session for this user")
		span.SetError(err, "creating new session for this user")
		return "", nil, err
	}

	questions, err := srv.storage.GetQuesions(ctx, topics)
	if err != nil {
		slog.Error(err.Error())
		span.SetError(err, "GetQuesions")
		return "", nil, errors.Wrap(err, "GetQuesions")
	}

	questionsMap := make(map[string]entities.Question, len(questions))
	for _, question := range questions {
		questionsMap[question.ID()] = question
	}

	if err = session.SetQuestions(
		questionsMap,
		time.Duration(len(questions))*srv.respondTime,
	); err != nil {
		slog.Error(err.Error())
		return "", nil, errors.Wrap(err, "SetQuestions")
	}

	if err := srv.storage.StoreSession(ctx, session); err != nil {
		slog.Error(err.Error())
		span.SetError(err, "StoreSession")
		return "", nil, errors.Wrap(err, "StoreSession")
	}

	slog.Info("CreateService completed")
	return session.GetSesionID(), questionsMap, nil
}

func (srv *SessionServiceBase) CompleteSession(
	ctx context.Context,
	sessionID string,
	answers []*entities.UserAnswer) (*entities.SessionResult, error) {
	slog.Info("CompleteSession started")
	ctx, span, cancel := tracing.GlobalTracer().Start(ctx, "CompleteSessionSpan")
	defer cancel()

	session, err := srv.storage.GetSessionBySessionID(ctx, sessionID)
	if err != nil {
		slog.Error(err.Error())
		span.SetError(err, "GetSessionBySessionID")
		return nil, errors.Wrap(err, "GetSessionBySessionID")
	}

	if err := session.SetUserAnswer(answers); err != nil {
		slog.Error(err.Error())
		span.SetError(err, "SetUserAnswer")
		return nil, errors.Wrap(err, "SetUserAnswer")
	}

	sessionResult, err := session.GetSessionResult()
	if err != nil {
		slog.Error(err.Error())
		span.SetError(err, "GetSessionResult")
		return nil, errors.Wrap(err, "GetSessionResult")
	}

	if err = srv.storage.StoreSession(ctx, session); err != nil {
		slog.Error(err.Error())
		span.SetError(err, "StoreSession")
		return nil, errors.Wrap(err, "StoreSession")
	}

	sessionResult.UserID = session.GetUserID()
	sessionResult.Topics = session.GetTopics()

	return sessionResult, nil
}

func (srv *SessionServiceBase) GetAllCompletedUserSessions(ctx context.Context, userID string) (
	[]*entities.Session, error) {
	slog.Info("GetAllCompletedUserSessions started")
	ctx, span, cancel := tracing.GlobalTracer().Start(ctx, "GetAllCompletedUserSessionsSpan")
	defer cancel()

	if userID == "" {
		err := errors.Wrap(entities.ErrInvalidParam, "userID not set")
		slog.Error(err.Error())
		span.SetError(err, "userID not set")
		return nil, err
	}

	sessions, err := srv.storage.GetAllCompletedUserSessions(ctx, userID)
	if err != nil {
		err = errors.Wrap(err, "get all completed user sessions failure")
		slog.Error(err.Error())
		span.SetError(err, "GetAllCompletedUserSessions")
		return nil, err
	}

	slog.Info("GetAllCompletedUserSessions completed")
	return sessions, nil
}

func (srv *SessionServiceBase) GetPassedStudentsTopics(ctx context.Context, students []string) (
	map[string][]*entities.Topic, error) {
	slog.Info("GetPassedStudentsTopics started")
	ctx, span, cancel := tracing.GlobalTracer().Start(ctx, "GetPassedStudentsTopicsSpan")
	defer cancel()

	passedTopics, err := srv.storage.GetPassedUserTopics(ctx, students)
	if err != nil {
		err = errors.Wrap(err, "GetPassedUserTopics failure")
		slog.Error(err.Error())
		span.SetError(err, "GetPassedUserTopics")
		return nil, err
	}

	slog.Info("GetPassedStudentsTopics completed")
	return passedTopics, nil
}
