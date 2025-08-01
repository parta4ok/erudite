package entities

import (
	"context"
	"time"

	"github.com/pkg/errors"
)

var (
	_ SessionState = (*ActiveSessionState)(nil)
)

type ActiveSessionState struct {
	holder    StateHolder
	questions map[string]Question
	startedAt time.Time
	duration  time.Duration
}

func NewActiveSessionState(questions map[string]Question, holder StateHolder,
	duration time.Duration, opts ...ActiveSessionStateOption) *ActiveSessionState {
	state := &ActiveSessionState{
		holder:    holder,
		questions: questions,
		startedAt: time.Now().UTC(),
		duration:  duration,
	}

	state.setOptions(opts...)

	return state
}

type ActiveSessionStateOption func(*ActiveSessionState)

func WithStartedAt(startedAt time.Time) ActiveSessionStateOption {
	return func(state *ActiveSessionState) {
		state.startedAt = startedAt
	}
}

func (state *ActiveSessionState) setOptions(opts ...ActiveSessionStateOption) {
	for _, opt := range opts {
		opt(state)
	}
}

func (state *ActiveSessionState) GetStatus() string {
	return ActiveState
}

func (state *ActiveSessionState) SetQuestions(_ map[string]Question,
	duration time.Duration) error {
	return errors.Wrapf(ErrInvalidState, "%s not support `SetQuestions`", state.GetStatus())
}

func (state *ActiveSessionState) SetUserAnswer(answers []*UserAnswer) error {
	isExpired := time.Now().UTC().After(state.startedAt.Add(state.duration))

	completedState := NewCompletedSessionState(state.questions, state.holder, answers,
		state.startedAt, isExpired)
	state.holder.ChangeState(completedState)

	return nil
}

func (state *ActiveSessionState) GetSessionResult() (*SessionResult, error) {
	return nil, errors.Wrapf(ErrInvalidState, "%s not support `GetSessionResult`", state.GetStatus())
}

func (state *ActiveSessionState) GetSessionDurationLimit() (time.Duration, error) {
	return state.duration, nil
}

func (state *ActiveSessionState) IsExpired() (bool, error) {
	return false, errors.Wrapf(
		ErrInvalidState, "%s not support `IsExpired`", state.GetStatus())
}

func (state *ActiveSessionState) GetQuestions() ([]Question, error) {
	questionsList := make([]Question, 0, len(state.questions))
	for _, question := range state.questions {
		questionsList = append(questionsList, question)
	}

	return questionsList, nil
}

func (state *ActiveSessionState) GetStartedAt() (time.Time, error) {
	return state.startedAt, nil
}

func (state *ActiveSessionState) GetUserAnswers() ([]*UserAnswer, error) {
	return nil, errors.Wrapf(
		ErrInvalidState, "%s not support `GetUserAnswers`", state.GetStatus())
}

func (state *ActiveSessionState) IsDailySessionLimitReached(ctx context.Context, userID string,
	topics []string) (bool, error) {
	return false, errors.Wrapf(
		ErrInvalidState, "%s not support `IsDailySessionLimitReached`", state.GetStatus())
}
