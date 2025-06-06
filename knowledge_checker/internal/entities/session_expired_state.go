package entities

import (
	"github.com/pkg/errors"
)

var (
	_ SessionState = (*ExpiredSessionState)(nil)
)

type ExpiredSessionState struct {
	session *Session
}

func NewExpiredSessionState() *ExpiredSessionState {
	session := &Session{}
	return &ExpiredSessionState{
		session: session,
	}
}

func (state *ExpiredSessionState) GetStatus() string {
	return ExpiredState
}

func (state *ExpiredSessionState) SetQuestions(_ *Session, _ map[uint64]Question) error {
	return errors.Wrapf(ErrInvalidState, "%s not support `SetQuetions`", state.GetStatus())
}

func (state *ExpiredSessionState) SetUserAnswer(_ *Session, _ []UserAnswer) error {
	return errors.Wrapf(ErrInvalidState, "%s not support `SetUserAnswer`", state.GetStatus())
}

func (state *ExpiredSessionState) SetSessionID(session *Session, sessessionID uint64) error {
	return errors.Wrapf(ErrInvalidState, "%s not support `SetSessionID`", state.GetStatus())
}

func (state *ExpiredSessionState) GetSessionResult(session *Session) (*SessionResult, error) {
	return &SessionResult{
		Done:           false,
		SuccessPercent: "0.0 session expired",
	}, nil
}
