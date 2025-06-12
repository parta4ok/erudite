package entities

import (
	"time"

	"github.com/pkg/errors"
)

const (
	DefaultTopicTimeLimit = time.Minute * 10
)

var (
	_ SessionState = (*InitSessionState)(nil)
)

type InitSessionState struct {
	stateHolder StateHolder
}

func NewInitSessionState(
	stateHolder StateHolder,
) *InitSessionState {

	return &InitSessionState{
		stateHolder: stateHolder,
	}
}

func (state *InitSessionState) GetStatus() string {
	return InitState
}

func (state *InitSessionState) SetQuestions(qestions map[uint64]Question,
	duration time.Duration) error {
	if len(qestions) == 0 {
		return errors.Wrap(ErrInvalidParam, "questions for selected topics not changed")
	}

	activeState := NewActiveSessionState(qestions, state.stateHolder, duration)
	state.stateHolder.ChangeState(activeState)

	return nil
}

func (state *InitSessionState) SetUserAnswer(_ []*UserAnswer) error {
	return errors.Wrapf(ErrInvalidState, "%s not support `SetUserAnswer`", state.GetStatus())
}

func (state *InitSessionState) GetSessionResult() (*SessionResult, error) {
	return nil, errors.Wrapf(ErrInvalidState, "%s not support `GetSessionResult`", state.GetStatus())
}
