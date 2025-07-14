package entities_test

import (
	"context"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"

	"github.com/parta4ok/kvs/question/internal/entities"
	"github.com/parta4ok/kvs/question/internal/entities/testdata"
)

func TestNewActiveSessionState(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuestion := testdata.NewMockQuestion(ctrl)
	questions := map[string]entities.Question{"1": mockQuestion}
	holder := testdata.NewMockStateHolder(ctrl)
	duration := time.Minute * 5

	state := entities.NewActiveSessionState(questions, holder, duration)

	require.NotNil(t, state)
	require.Equal(t, entities.ActiveState, state.GetStatus())
}

func TestActiveSessionState_WithStartedAt(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuestion := testdata.NewMockQuestion(ctrl)
	questions := map[string]entities.Question{"1": mockQuestion}
	holder := testdata.NewMockStateHolder(ctrl)
	duration := time.Minute * 5
	startTime := time.Now()

	state := entities.NewActiveSessionState(questions, holder, duration,
		entities.WithStartedAt(startTime))

	require.NotNil(t, state)
	result, err := state.GetStartedAt()
	require.NoError(t, err)
	require.Equal(t, startTime, result)
}

func TestActiveSessionState_SetQuestions(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuestion := testdata.NewMockQuestion(ctrl)
	questions := map[string]entities.Question{"1": mockQuestion}
	holder := testdata.NewMockStateHolder(ctrl)
	duration := time.Minute * 5

	state := entities.NewActiveSessionState(questions, holder, duration)

	err := state.SetQuestions(questions, duration)

	require.Error(t, err)
	require.Contains(t, err.Error(), "not support `SetQuestions`")
}

func TestActiveSessionState_SetUserAnswer(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuestion := testdata.NewMockQuestion(ctrl)
	questions := map[string]entities.Question{"1": mockQuestion}
	holder := testdata.NewMockStateHolder(ctrl)
	duration := time.Minute * 5

	state := entities.NewActiveSessionState(questions, holder, duration)

	userAnswer, err := entities.NewUserAnswer("1", []string{"answer"})
	require.NoError(t, err)

	holder.EXPECT().ChangeState(gomock.Any()).Times(1)

	err = state.SetUserAnswer([]*entities.UserAnswer{userAnswer})

	require.NoError(t, err)
}

func TestActiveSessionState_GetSessionResult(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuestion := testdata.NewMockQuestion(ctrl)
	questions := map[string]entities.Question{"1": mockQuestion}
	holder := testdata.NewMockStateHolder(ctrl)
	duration := time.Minute * 5

	state := entities.NewActiveSessionState(questions, holder, duration)

	result, err := state.GetSessionResult()

	require.Error(t, err)
	require.Nil(t, result)
	require.Contains(t, err.Error(), "not support `GetSessionResult`")
}

func TestActiveSessionState_GetSessionDurationLimit(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuestion := testdata.NewMockQuestion(ctrl)
	questions := map[string]entities.Question{"1": mockQuestion}
	holder := testdata.NewMockStateHolder(ctrl)
	duration := time.Minute * 5

	state := entities.NewActiveSessionState(questions, holder, duration)

	result, err := state.GetSessionDurationLimit()

	require.NoError(t, err)
	require.Equal(t, duration, result)
}

func TestActiveSessionState_IsExpired_NotSupported(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuestion := testdata.NewMockQuestion(ctrl)
	questions := map[string]entities.Question{"1": mockQuestion}
	holder := testdata.NewMockStateHolder(ctrl)
	duration := time.Hour

	state := entities.NewActiveSessionState(questions, holder, duration)

	isExpired, err := state.IsExpired()

	require.Error(t, err)
	require.False(t, isExpired)
	require.Contains(t, err.Error(), "not support `IsExpired`")
}

func TestActiveSessionState_GetQuestions(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuestion := testdata.NewMockQuestion(ctrl)
	questions := map[string]entities.Question{"1": mockQuestion}
	holder := testdata.NewMockStateHolder(ctrl)
	duration := time.Minute * 5

	state := entities.NewActiveSessionState(questions, holder, duration)

	result, err := state.GetQuestions()

	require.NoError(t, err)
	require.Len(t, result, 1)
	require.Equal(t, mockQuestion, result[0])
}

func TestActiveSessionState_GetUserAnswers_NotSupported(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuestion := testdata.NewMockQuestion(ctrl)
	questions := map[string]entities.Question{"1": mockQuestion}
	holder := testdata.NewMockStateHolder(ctrl)
	duration := time.Minute * 5

	state := entities.NewActiveSessionState(questions, holder, duration)

	result, err := state.GetUserAnswers()

	require.Error(t, err)
	require.Nil(t, result)
	require.Contains(t, err.Error(), "not support `GetUserAnswers`")
}

func TestActiveSessionState_IsDailySessionLimitReached(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuestion := testdata.NewMockQuestion(ctrl)
	questions := map[string]entities.Question{"1": mockQuestion}
	holder := testdata.NewMockStateHolder(ctrl)
	duration := time.Minute * 5

	state := entities.NewActiveSessionState(questions, holder, duration)

	result, err := state.IsDailySessionLimitReached(context.TODO(), "1", []string{"topic"})

	require.Error(t, err)
	require.False(t, result)
	require.Contains(t, err.Error(), "not support `IsDailySessionLimitReached`")
}
