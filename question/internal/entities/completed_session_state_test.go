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

func TestNewCompletedSessionState(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuestion := testdata.NewMockQuestion(ctrl)
	questions := map[string]entities.Question{"1": mockQuestion}
	holder := testdata.NewMockStateHolder(ctrl)
	userAnswer, err := entities.NewUserAnswer("1", []string{"answer"})
	require.NoError(t, err)
	answers := []*entities.UserAnswer{userAnswer}
	isExpired := false
	startedAt := time.Now()

	state := entities.NewCompletedSessionState(questions, holder, answers, startedAt, isExpired)

	require.NotNil(t, state)
	require.Equal(t, entities.CompletedState, state.GetStatus())
}

func TestCompletedSessionState_SetQuestions(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuestion := testdata.NewMockQuestion(ctrl)
	questions := map[string]entities.Question{"1": mockQuestion}
	holder := testdata.NewMockStateHolder(ctrl)
	answers := []*entities.UserAnswer{}
	startedAt := time.Now()

	state := entities.NewCompletedSessionState(questions, holder, answers, startedAt, false)

	err := state.SetQuestions(questions, 0)

	require.Error(t, err)
	require.Contains(t, err.Error(), "not support `SetQuestions`")
}

func TestCompletedSessionState_SetUserAnswer(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuestion := testdata.NewMockQuestion(ctrl)
	questions := map[string]entities.Question{"1": mockQuestion}
	holder := testdata.NewMockStateHolder(ctrl)
	answers := []*entities.UserAnswer{}
	startedAt := time.Now()

	state := entities.NewCompletedSessionState(questions, holder, answers, startedAt, false)

	userAnswer, err := entities.NewUserAnswer("1", []string{"answer"})
	require.NoError(t, err)

	err = state.SetUserAnswer([]*entities.UserAnswer{userAnswer})

	require.Error(t, err)
	require.Contains(t, err.Error(), "not support `SetUserAnswer`")
}

func TestCompletedSessionState_GetSessionResult_Expired(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuestion := testdata.NewMockQuestion(ctrl)
	questions := map[string]entities.Question{"1": mockQuestion}
	holder := testdata.NewMockStateHolder(ctrl)
	answers := []*entities.UserAnswer{}
	isExpired := true
	startedAt := time.Now()

	state := entities.NewCompletedSessionState(questions, holder, answers, startedAt, isExpired)

	result, err := state.GetSessionResult()

	require.NoError(t, err)
	require.NotNil(t, result)
	require.False(t, result.IsSuccess)
	require.Equal(t, "session expired", result.Grade)
}

func TestCompletedSessionState_GetSessionResult_Success(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuestion := testdata.NewMockQuestion(ctrl)
	mockQuestion.EXPECT().ID().Return("1").AnyTimes()
	mockQuestion.EXPECT().IsAnswerCorrect(gomock.Any()).Return(true).Times(1)

	questions := map[string]entities.Question{"1": mockQuestion}
	holder := testdata.NewMockStateHolder(ctrl)

	userAnswer, err := entities.NewUserAnswer("1", []string{"correct"})
	require.NoError(t, err)
	answers := []*entities.UserAnswer{userAnswer}
	startedAt := time.Now()

	state := entities.NewCompletedSessionState(questions, holder, answers, startedAt, false)

	result, err := state.GetSessionResult()

	require.NoError(t, err)
	require.NotNil(t, result)
	require.True(t, result.IsSuccess)
	require.Equal(t, "100.00 percents", result.Grade)
}

func TestCompletedSessionState_GetSessionResult_Failure(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuestion := testdata.NewMockQuestion(ctrl)
	mockQuestion.EXPECT().ID().Return("1").AnyTimes()
	mockQuestion.EXPECT().IsAnswerCorrect(gomock.Any()).Return(false).Times(1)

	questions := map[string]entities.Question{"1": mockQuestion}
	holder := testdata.NewMockStateHolder(ctrl)

	userAnswer, err := entities.NewUserAnswer("1", []string{"wrong"})
	require.NoError(t, err)
	answers := []*entities.UserAnswer{userAnswer}
	startedAt := time.Now()

	state := entities.NewCompletedSessionState(questions, holder, answers, startedAt, false)

	result, err := state.GetSessionResult()

	require.NoError(t, err)
	require.NotNil(t, result)
	require.False(t, result.IsSuccess)
	require.Equal(t, "0.00 percents", result.Grade)
}

func TestCompletedSessionState_GetSessionDurationLimit(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuestion := testdata.NewMockQuestion(ctrl)
	questions := map[string]entities.Question{"1": mockQuestion}
	holder := testdata.NewMockStateHolder(ctrl)
	answers := []*entities.UserAnswer{}
	startedAt := time.Now()

	state := entities.NewCompletedSessionState(questions, holder, answers, startedAt, false)

	result, err := state.GetSessionDurationLimit()

	require.Error(t, err)
	require.Equal(t, int64(0), result.Nanoseconds())
	require.Contains(t, err.Error(), "not support `GetSessionDurationLimit`")
}

func TestCompletedSessionState_IsExpired(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuestion := testdata.NewMockQuestion(ctrl)
	questions := map[string]entities.Question{"1": mockQuestion}
	holder := testdata.NewMockStateHolder(ctrl)
	answers := []*entities.UserAnswer{}
	startedAt := time.Now()

	state := entities.NewCompletedSessionState(questions, holder, answers, startedAt, true)

	isExpired, err := state.IsExpired()

	require.NoError(t, err)
	require.True(t, isExpired)
}

func TestCompletedSessionState_GetQuestions(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuestion := testdata.NewMockQuestion(ctrl)
	questions := map[string]entities.Question{"1": mockQuestion}
	holder := testdata.NewMockStateHolder(ctrl)
	answers := []*entities.UserAnswer{}
	startedAt := time.Now()

	state := entities.NewCompletedSessionState(questions, holder, answers, startedAt, false)

	result, err := state.GetQuestions()

	require.NoError(t, err)
	require.Len(t, result, 1)
	require.Equal(t, mockQuestion, result[0])
}

func TestCompletedSessionState_GetStartedAt(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuestion := testdata.NewMockQuestion(ctrl)
	questions := map[string]entities.Question{"1": mockQuestion}
	holder := testdata.NewMockStateHolder(ctrl)
	answers := []*entities.UserAnswer{}
	startedAt := time.Now()

	state := entities.NewCompletedSessionState(questions, holder, answers, startedAt, false)

	result, err := state.GetStartedAt()

	require.Equal(t, startedAt, result)
	require.NoError(t, err)
}

func TestCompletedSessionState_GetUserAnswers(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuestion := testdata.NewMockQuestion(ctrl)
	questions := map[string]entities.Question{"1": mockQuestion}
	holder := testdata.NewMockStateHolder(ctrl)

	userAnswer, err := entities.NewUserAnswer("1", []string{"answer"})
	require.NoError(t, err)
	answers := []*entities.UserAnswer{userAnswer}
	startedAt := time.Now()

	state := entities.NewCompletedSessionState(questions, holder, answers, startedAt, false)

	result, err := state.GetUserAnswers()

	require.NoError(t, err)
	require.Len(t, result, 1)
	require.Equal(t, userAnswer, result[0])
}

func TestCompletedSessionState_IsDailySessionLimitReached(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuestion := testdata.NewMockQuestion(ctrl)
	questions := map[string]entities.Question{"1": mockQuestion}
	holder := testdata.NewMockStateHolder(ctrl)
	answers := []*entities.UserAnswer{}
	startedAt := time.Now()

	state := entities.NewCompletedSessionState(questions, holder, answers, startedAt, false)

	result, err := state.IsDailySessionLimitReached(context.TODO(), "1", []string{"topic"})

	require.Error(t, err)
	require.False(t, result)
	require.Contains(t, err.Error(), "not support `IsDailySessionLimitReached`")
}

func TestCompletedSessionState_GetSessionResult_InvalidQuestionType(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockQuestion := testdata.NewMockQuestion(ctrl)
	questions := map[string]entities.Question{"1": mockQuestion}
	holder := testdata.NewMockStateHolder(ctrl)
	answer, err := entities.NewUserAnswer("2", []string{"answer"})
	require.NoError(t, err)

	answers := []*entities.UserAnswer{answer}
	startedAt := time.Now()

	state := entities.NewCompletedSessionState(questions, holder, answers, startedAt, false)

	result, err := state.GetSessionResult()
	require.ErrorIs(t, err, entities.ErrInvalidParam)
	require.Nil(t, result)
}
