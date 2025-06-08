package entities

import (
	"time"

	"github.com/pkg/errors"
)

var (
	_ SessionState = (*ActiveSessionState)(nil)
)

type ActiveSessionState struct {
	holder    StateHolder
	questions map[uint64]Question
	startedAt time.Time
	duration  time.Duration
}

func NewActiveSessionState(questions map[uint64]Question, holder StateHolder, duration time.Duration) *ActiveSessionState {
	state := &ActiveSessionState{
		holder:    holder,
		questions: questions,
		startedAt: time.Now().UTC(),
		duration:  duration,
	}

	return state
}

func (state *ActiveSessionState) GetStatus() string {
	return ActiveState
}

func (state *ActiveSessionState) SetQuestions(qestions map[uint64]Question, duration time.Duration) error {
	return errors.Wrapf(ErrInvalidState, "%s not support `SetQuestions`", state.GetStatus())
}

func (state *ActiveSessionState) SetUserAnswer(answers []UserAnswer) error {
	isExpired := time.Now().UTC().After(state.startedAt.Add(state.duration))

	completedState := NewCompletedSessionState(state.questions, state.holder, answers, isExpired)
	state.holder.ChangeState(completedState)

	return nil
}

func (state *ActiveSessionState) GetSessionResult() (*SessionResult, error) {
	return nil, errors.Wrapf(ErrInvalidState, "%s not support `GetSessionResult`", state.GetStatus())
}
