package entities

import (
	"context"
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
	stateHolder    StateHolder
	sessionStorage SessionStorage
}

func NewInitSessionState(stateHolder StateHolder, sessionStorage SessionStorage) *InitSessionState {

	return &InitSessionState{
		stateHolder:    stateHolder,
		sessionStorage: sessionStorage,
	}
}

func (state *InitSessionState) GetStatus() string {
	return InitState
}

func (state *InitSessionState) SetQuestions(qestions map[string]Question,
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

func (state *InitSessionState) GetSessionDurationLimit() (time.Duration, error) {
	return time.Duration(0), errors.Wrapf(
		ErrInvalidState, "%s not support `GetSessionDurationLimit`", state.GetStatus())
}

func (state *InitSessionState) IsExpired() (bool, error) {
	return false, errors.Wrapf(
		ErrInvalidState, "%s not support `IsExpired`", state.GetStatus())
}

func (state *InitSessionState) GetQuestions() ([]Question, error) {
	return nil, errors.Wrapf(
		ErrInvalidState, "%s not support `GetQuestions`", state.GetStatus())
}

func (state *InitSessionState) GetStartedAt() (time.Time, error) {
	return time.Time{}, errors.Wrapf(
		ErrInvalidState, "%s not support `GetStartedAt`", state.GetStatus())
}

func (state *InitSessionState) GetUserAnswers() ([]*UserAnswer, error) {
	return nil, errors.Wrapf(
		ErrInvalidState, "%s not support `GetUserAnswers`", state.GetStatus())
}

func (state *InitSessionState) IsDailySessionLimitReached(ctx context.Context,
	userID string, topics []string) (bool, error) {
	return state.sessionStorage.IsDailySessionLimitReached(ctx, userID, topics)
}
