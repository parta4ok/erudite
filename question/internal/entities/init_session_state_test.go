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

func TestNewInitSessionState(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	holder := testdata.NewMockStateHolder(ctrl)
	storage := testdata.NewMockSessionStorage(ctrl)

	state := entities.NewInitSessionState(holder, storage)

	require.NotNil(t, state)
	require.Equal(t, entities.InitState, state.GetStatus())
}

func TestInitSessionState_SetQuestions_Success(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	holder := testdata.NewMockStateHolder(ctrl)
	storage := testdata.NewMockSessionStorage(ctrl)
	mockQuestion := testdata.NewMockQuestion(ctrl)
	questions := map[uint64]entities.Question{1: mockQuestion}
	duration := time.Minute * 5

	state := entities.NewInitSessionState(holder, storage)

	holder.EXPECT().ChangeState(gomock.Any()).Times(1)

	err := state.SetQuestions(questions, duration)

	require.NoError(t, err)
}

func TestInitSessionState_SetQuestions_EmptyQuestions(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	holder := testdata.NewMockStateHolder(ctrl)
	storage := testdata.NewMockSessionStorage(ctrl)
	questions := map[uint64]entities.Question{}
	duration := time.Minute * 5

	state := entities.NewInitSessionState(holder, storage)

	err := state.SetQuestions(questions, duration)

	require.Error(t, err)
	require.Contains(t, err.Error(), "questions for selected topics not changed")
}

func TestInitSessionState_SetUserAnswer(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	holder := testdata.NewMockStateHolder(ctrl)
	storage := testdata.NewMockSessionStorage(ctrl)

	state := entities.NewInitSessionState(holder, storage)

	userAnswer, err := entities.NewUserAnswer(1, []string{"answer"})
	require.NoError(t, err)

	err = state.SetUserAnswer([]*entities.UserAnswer{userAnswer})

	require.Error(t, err)
	require.Contains(t, err.Error(), "not support `SetUserAnswer`")
}

func TestInitSessionState_GetSessionResult(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	holder := testdata.NewMockStateHolder(ctrl)
	storage := testdata.NewMockSessionStorage(ctrl)

	state := entities.NewInitSessionState(holder, storage)

	result, err := state.GetSessionResult()

	require.Error(t, err)
	require.Nil(t, result)
	require.Contains(t, err.Error(), "not support `GetSessionResult`")
}

func TestInitSessionState_GetSessionDurationLimit(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	holder := testdata.NewMockStateHolder(ctrl)
	storage := testdata.NewMockSessionStorage(ctrl)

	state := entities.NewInitSessionState(holder, storage)

	result, err := state.GetSessionDurationLimit()

	require.Error(t, err)
	require.Equal(t, int64(0), result.Nanoseconds())
	require.Contains(t, err.Error(), "not support `GetSessionDurationLimit`")
}

func TestInitSessionState_IsExpired(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	holder := testdata.NewMockStateHolder(ctrl)
	storage := testdata.NewMockSessionStorage(ctrl)

	state := entities.NewInitSessionState(holder, storage)

	isExpired, err := state.IsExpired()

	require.Error(t, err)
	require.False(t, isExpired)
	require.Contains(t, err.Error(), "not support `IsExpired`")
}

func TestInitSessionState_GetQuestions(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	holder := testdata.NewMockStateHolder(ctrl)
	storage := testdata.NewMockSessionStorage(ctrl)

	state := entities.NewInitSessionState(holder, storage)

	result, err := state.GetQuestions()

	require.Error(t, err)
	require.Nil(t, result)
	require.Contains(t, err.Error(), "not support `GetQuestions`")
}

func TestInitSessionState_GetStartedAt(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	holder := testdata.NewMockStateHolder(ctrl)
	storage := testdata.NewMockSessionStorage(ctrl)

	state := entities.NewInitSessionState(holder, storage)

	result, err := state.GetStartedAt()

	require.Error(t, err)
	require.True(t, result.IsZero())
	require.Contains(t, err.Error(), "not support `GetStartedAt`")
}

func TestInitSessionState_GetUserAnswers(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	holder := testdata.NewMockStateHolder(ctrl)
	storage := testdata.NewMockSessionStorage(ctrl)

	state := entities.NewInitSessionState(holder, storage)

	result, err := state.GetUserAnswers()

	require.Error(t, err)
	require.Nil(t, result)
	require.Contains(t, err.Error(), "not support `GetUserAnswers`")
}

func TestInitSessionState_IsDailySessionLimitReached_Success(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	holder := testdata.NewMockStateHolder(ctrl)
	storage := testdata.NewMockSessionStorage(ctrl)
	ctx := context.Background()
	userID := uint64(1)
	topics := []string{"topic1"}

	state := entities.NewInitSessionState(holder, storage)

	storage.EXPECT().IsDailySessionLimitReached(ctx, userID, topics).Return(false, nil).Times(1)

	result, err := state.IsDailySessionLimitReached(ctx, userID, topics)

	require.NoError(t, err)
	require.False(t, result)
}

func TestInitSessionState_IsDailySessionLimitReached_LimitReached(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	holder := testdata.NewMockStateHolder(ctrl)
	storage := testdata.NewMockSessionStorage(ctrl)
	ctx := context.Background()
	userID := uint64(1)
	topics := []string{"topic1"}

	state := entities.NewInitSessionState(holder, storage)

	storage.EXPECT().IsDailySessionLimitReached(ctx, userID, topics).Return(true, nil).Times(1)

	result, err := state.IsDailySessionLimitReached(ctx, userID, topics)

	require.NoError(t, err)
	require.True(t, result)
}
