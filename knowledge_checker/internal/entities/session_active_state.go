package entities

import (
	"time"

	"github.com/pkg/errors"
)

var (
	_ SessionState = (*ActiveSessionState)(nil)
)

type ActiveSessionState struct {
	session *Session
}

func NewActiveSessionState(userID uint64, topics []string, duration time.Duration,
	questions map[uint64]Question) *ActiveSessionState {
	session := &Session{
		userID:    userID,
		topics:    topics,
		duration:  duration,
		questions: questions,
		startedAt: time.Now().UTC(),
	}
	return &ActiveSessionState{
		session: session,
	}
}

func (state *ActiveSessionState) GetStatus() string {
	return InitState
}

func (state *ActiveSessionState) SetQuestions(_ *Session, _ map[uint64]Question) error {
	return errors.Wrapf(ErrInvalidState, "%s not support `SetQuetions`", state.GetStatus())
}

func (state *ActiveSessionState) SetUserAnswer(session *Session, answers []UserAnswer) error {
	if time.Now().UTC().After(session.startedAt.Add(session.duration)) {
		session.state = NewExpiredSessionState()
		return nil
	}

	session.state = NewCompletedSessionState(
		session.userID,
		session.sessionID,
		session.topics,
		session.questions,
		session.answers,
	)

	return nil
}

func (state *ActiveSessionState) SetSessionID(session *Session, sessessionID uint64) error {
	return errors.Wrapf(ErrInvalidState, "%s not support `SetSessionID`", state.GetStatus())
}

func (state *ActiveSessionState) GetSessionResult(session *Session) (*SessionResult, error) {
	return nil, errors.Wrapf(ErrInvalidState, "%s not support `GetSessionResult`", state.GetStatus())
}
