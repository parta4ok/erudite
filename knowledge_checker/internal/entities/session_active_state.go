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

func NewActiveSessionState(session *Session) *ActiveSessionState {
	return &ActiveSessionState{
		session: session,
	}
}

func (state *ActiveSessionState) GetStatus() string {
	return ActiveState
}

func (state *ActiveSessionState) SetQuestions(_ *Session, _ map[uint64]Question) error {
	return errors.Wrapf(ErrInvalidState, "%s not support `SetQuetions`", state.GetStatus())
}

func (state *ActiveSessionState) SetUserAnswer(session *Session, answers []UserAnswer) error {
	if time.Now().UTC().After(session.startedAt.Add(session.duration)) {
		session.state = NewExpiredSessionState()
		return nil
	}

	session.answers = append(session.answers, answers...)
	session.state = NewCompletedSessionState(session)

	return nil
}

func (state *ActiveSessionState) SetSessionID(session *Session, sessessionID uint64) error {
	return errors.Wrapf(ErrInvalidState, "%s not support `SetSessionID`", state.GetStatus())
}

func (state *ActiveSessionState) GetSessionResult(session *Session) (*SessionResult, error) {
	return nil, errors.Wrapf(ErrInvalidState, "%s not support `GetSessionResult`", state.GetStatus())
}
