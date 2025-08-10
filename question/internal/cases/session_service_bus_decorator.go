package cases

import (
	"context"
	"log/slog"
	"time"

	"github.com/parta4ok/kvs/question/internal/entities"
	"github.com/pkg/errors"
)

// SessionServiceBusDecorator is a decorator for SessionService that adds message broker
// functionality.
type SessionServiceBusDecorator struct {
	sessionService SessionService
	messageBroker  MessageBroker
}

func NewSessionServiceBusDecorator(sessionService SessionService, messageBroker MessageBroker) (
	*SessionServiceBusDecorator, error) {
	if sessionService == nil {
		return nil, errors.Wrap(entities.ErrInvalidParam, "base session service not set")
	}
	if messageBroker == nil {
		return nil, errors.Wrap(entities.ErrInvalidParam, "message broker not set")
	}

	return &SessionServiceBusDecorator{
		sessionService: sessionService,
		messageBroker:  messageBroker,
	}, nil
}

func (service *SessionServiceBusDecorator) CompleteSession(ctx context.Context, sessionID string,
	answers []*entities.UserAnswer) (
	*entities.SessionResult, error) {
	slog.Info("CompleteSession in SessionServiceBusDecorator started")
	sessionResult, err := service.sessionService.CompleteSession(ctx, sessionID, answers)
	if err != nil {
		err = errors.Wrap(err, "CompleteSession in SessionServiceBusDecorator")
		slog.Error(err.Error())
		return nil, err
	}

	go func() {
		msgCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := service.messageBroker.SessionFinishedEvent(msgCtx, sessionResult); err != nil {
			err = errors.Wrap(err, "SessionFinishedEvent in SessionServiceBusDecorator")
			slog.Warn("Failed to send session finished event",
				"session_id", sessionID,
				"error", err)
		}
	}()

	slog.Info("CompleteSession in SessionServiceBusDecorator completed")
	return sessionResult, nil
}

func (service *SessionServiceBusDecorator) CreateSession(ctx context.Context, userID string,
	topics []string) (
	string, map[string]entities.Question, error) {
	slog.Info("CreateSession in SessionServiceBusDecorator started")
	sessionID, questions, err := service.sessionService.CreateSession(ctx, userID, topics)
	if err != nil {
		err = errors.Wrap(err, "CreateSession in SessionServiceBusDecorator")
		slog.Error(err.Error())
		return "", nil, err
	}

	slog.Info("CreateSession in SessionServiceBusDecorator completed")
	return sessionID, questions, nil
}

func (service *SessionServiceBusDecorator) GetAllCompletedUserSessions(ctx context.Context,
	userID string) ([]*entities.Session, error) {
	slog.Info("GetAllCompletedUserSessions in SessionServiceBusDecorator started")
	sessions, err := service.sessionService.GetAllCompletedUserSessions(ctx, userID)
	if err != nil {
		err = errors.Wrap(err, "GetAllCompletedUserSessions in SessionServiceBusDecorator")
		slog.Error(err.Error())
		return nil, err
	}

	slog.Info("GetAllCompletedUserSessions in SessionServiceBusDecorator completed")
	return sessions, nil
}

func (service *SessionServiceBusDecorator) ShowTopics(ctx context.Context) ([]string, error) {
	slog.Info("ShowTopics in SessionServiceBusDecorator started")
	topics, err := service.sessionService.ShowTopics(ctx)
	if err != nil {
		err = errors.Wrap(err, "ShowTopics in SessionServiceBusDecorator")
		slog.Error(err.Error())
		return nil, err
	}

	slog.Info("ShowTopics in SessionServiceBusDecorator completed")
	return topics, nil
}
