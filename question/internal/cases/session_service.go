package cases

import (
	"context"
	"log/slog"
	"time"

	"github.com/pkg/errors"

	"github.com/parta4ok/kvs/question/internal/entities"
)

const (
	defaultTopicDuration = time.Minute * 10
)

type SessionService struct {
	storage        Storage
	sessionStorage entities.SessionStorage
	generator      entities.IDGenerator
	topicDuration  time.Duration
}

func NewSessionService(storage Storage, sessionStorage entities.SessionStorage,
	generator entities.IDGenerator, opts ...SessionServiceOption) (*SessionService, error) {
	if storage == nil {
		return nil, errors.Wrapf(entities.ErrInvalidParam, "storage not set")
	}

	if sessionStorage == nil {
		return nil, errors.Wrapf(entities.ErrInvalidParam, "session storage not set")
	}

	if generator == nil {
		return nil, errors.Wrapf(entities.ErrInvalidParam, "generator not set")
	}

	service := &SessionService{
		storage:        storage,
		sessionStorage: sessionStorage,
		generator:      generator,
		topicDuration:  defaultTopicDuration,
	}

	service.setOptions(opts...)

	return service, nil
}

type SessionServiceOption func(*SessionService)

func WithCustomSessionDuration(dur time.Duration) SessionServiceOption {
	return func(srv *SessionService) {
		srv.topicDuration = dur
	}
}

func (srv *SessionService) setOptions(opts ...SessionServiceOption) {
	for _, opt := range opts {
		opt(srv)
	}
}

func (srv *SessionService) ShowTopics(ctx context.Context) ([]string, error) {
	slog.Info("ShowTopics started")

	topics, err := srv.storage.GetTopics(ctx)
	if err != nil {
		slog.Error(err.Error())
		return nil, errors.Wrap(err, "GetTopics")
	}

	slog.Info("ShowTopics completed")
	return topics, nil
}

func (srv *SessionService) CreateSession(ctx context.Context, userID string,
	topics []string) (string, map[string]entities.Question, error) {
	slog.Info("CreateSession started")

	session, err := entities.NewSession(userID, topics, srv.generator, srv.sessionStorage)
	if err != nil {
		slog.Error(err.Error())
		return "", nil, errors.Wrap(err, "NewSession")
	}

	forbidded, err := session.IsDailySessionLimitReached(ctx, userID, topics)
	if err != nil {
		slog.Error(err.Error())
		return "", nil, errors.Wrap(err, "IsDailySessionLimitReached")
	}

	if forbidded {
		return "", nil, errors.Wrap(entities.ErrForbidden, "creating new session for this user")
	}

	questions, err := srv.storage.GetQuesions(ctx, topics)
	if err != nil {
		slog.Error(err.Error())
		return "", nil, errors.Wrap(err, "GetQuesions")
	}

	questionsMap := make(map[string]entities.Question, len(questions))
	for _, question := range questions {
		questionsMap[question.ID()] = question
	}

	if err = session.SetQuestions(questionsMap, srv.topicDuration); err != nil {
		slog.Error(err.Error())
		return "", nil, errors.Wrap(err, "SetQuestions")
	}

	if err := srv.storage.StoreSession(ctx, session); err != nil {
		slog.Error(err.Error())
		return "", nil, errors.Wrap(err, "StoreSession")
	}

	slog.Info("CreateService completed")
	return session.GetSesionID(), questionsMap, nil
}

func (srv *SessionService) CompleteSession(
	ctx context.Context,
	sessionID string,
	answers []*entities.UserAnswer) (*entities.SessionResult, error) {
	slog.Info("CompleteSession started")

	session, err := srv.storage.GetSessionBySessionID(ctx, sessionID)
	if err != nil {
		slog.Error(err.Error())
		return nil, errors.Wrap(err, "GetSessionBySessionID")
	}

	if err := session.SetUserAnswer(answers); err != nil {
		slog.Error(err.Error())
		return nil, errors.Wrap(err, "SetUserAnswer")
	}

	sessionResult, err := session.GetSessionResult()
	if err != nil {
		slog.Error(err.Error())
		return nil, errors.Wrap(err, "GetSessionResult")
	}

	if err = srv.storage.StoreSession(ctx, session); err != nil {
		slog.Error(err.Error())
		return nil, errors.Wrap(err, "StoreSession")
	}

	return sessionResult, nil
}

func (srv *SessionService) GetAllCompletedUserSessions(ctx context.Context, userID string) (
	[]*entities.Session, error) {
	slog.Info("GetAllCompletedUserSessions started")

	if userID == "" {
		err := errors.Wrap(entities.ErrInvalidParam, "userID not set")
		slog.Error(err.Error())
		return nil, err
	}

	sessions, err := srv.storage.GetAllCompletedUserSessions(ctx, userID)
	if err != nil {
		err = errors.Wrap(err, "get all completed user sessions failure")
		slog.Error(err.Error())
		return nil, err
	}

	slog.Info("GetAllCompletedUserSessions completed")
	return sessions, nil
}
