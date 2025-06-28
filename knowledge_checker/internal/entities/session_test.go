package entities_test

import (
	"context"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"

	"github.com/parta4ok/kvs/knowledge_checker/internal/entities"
	"github.com/parta4ok/kvs/knowledge_checker/internal/entities/testdata"
)

func TestNewSession(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer func() {
		t.Cleanup(ctrl.Finish)
	}()
	ctx := context.TODO()
	userID := uint64(1)
	topics := []string{"1"}

	generator := testdata.NewMockIDGenerator(ctrl)
	generator.EXPECT().GenerateID().Return(uint64(2))
	storage := testdata.NewMockSessionStorage(ctrl)
	storage.EXPECT().IsDailySessionLimitReached(ctx, userID, topics).Return(false, nil)

	question := testdata.NewMockQuestion(ctrl)

	session, err := entities.NewSession(uint64(userID), topics, generator, storage)
	require.NoError(t, err)
	require.NotNil(t, session)

	require.Equal(t, entities.InitState, session.GetStatus())

	forbidden, err := session.IsDailySessionLimitReached(ctx, uint64(userID), topics)
	require.False(t, forbidden)
	require.NoError(t, err)

	quesionMap := map[uint64]entities.Question{3: question}

	err = session.SetQuestions(quesionMap, time.Second*30)
	require.NoError(t, err)

	require.Equal(t, entities.ActiveState, session.GetStatus())

	answer, err := entities.NewUserAnswer(uint64(userID), []string{"random answer"})
	require.NoError(t, err)

	err = session.SetUserAnswer([]*entities.UserAnswer{answer})
	require.NoError(t, err)

	require.Equal(t, entities.CompletedState, session.GetStatus())
}

func TestSession_WithSessionID(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	userID := uint64(1)
	topics := []string{"topic1"}
	sessionID := uint64(123)
	generator := testdata.NewMockIDGenerator(ctrl)
	generator.EXPECT().GenerateID().Return(uint64(999))
	storage := testdata.NewMockSessionStorage(ctrl)

	session, err := entities.NewSession(userID, topics, generator, storage,
		entities.WithSessionID(sessionID))

	require.NoError(t, err)
	require.NotNil(t, session)
	require.Equal(t, sessionID, session.GetSesionID())
}

func TestSession_WithNilState(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	userID := uint64(1)
	topics := []string{"topic1"}
	generator := testdata.NewMockIDGenerator(ctrl)
	generator.EXPECT().GenerateID().Return(uint64(1))
	storage := testdata.NewMockSessionStorage(ctrl)

	session, err := entities.NewSession(userID, topics, generator, storage,
		entities.WithNilState())

	require.NoError(t, err)
	require.NotNil(t, session)
}

func TestNewSessionWithCustomState(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	userID := uint64(1)
	sessionID := uint64(123)
	topics := []string{"topic1"}
	state := testdata.NewMockSessionState(ctrl)

	session := entities.NewSessionWithCustomState(sessionID, userID, topics, state)

	require.NotNil(t, session)
	require.Equal(t, userID, session.GetUserID())
	require.Equal(t, sessionID, session.GetSesionID())
	require.Equal(t, topics, session.GetTopics())
}

func TestSession_GetUserID(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	userID := uint64(42)
	topics := []string{"topic1"}
	generator := testdata.NewMockIDGenerator(ctrl)
	generator.EXPECT().GenerateID().Return(uint64(1))
	storage := testdata.NewMockSessionStorage(ctrl)

	session, err := entities.NewSession(userID, topics, generator, storage)
	require.NoError(t, err)

	result := session.GetUserID()

	require.Equal(t, userID, result)
}

func TestSession_GetTopics(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	userID := uint64(1)
	topics := []string{"Go", "Databases"}
	generator := testdata.NewMockIDGenerator(ctrl)
	generator.EXPECT().GenerateID().Return(uint64(1))
	storage := testdata.NewMockSessionStorage(ctrl)

	session, err := entities.NewSession(userID, topics, generator, storage)
	require.NoError(t, err)

	result := session.GetTopics()

	require.Equal(t, topics, result)
}

func TestSession_GetSessionResult(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	userID := uint64(1)
	topics := []string{"topic1"}
	generator := testdata.NewMockIDGenerator(ctrl)
	generator.EXPECT().GenerateID().Return(uint64(1))
	storage := testdata.NewMockSessionStorage(ctrl)

	session, err := entities.NewSession(userID, topics, generator, storage)
	require.NoError(t, err)

	mockState := testdata.NewMockSessionState(ctrl)
	expectedResult := &entities.SessionResult{IsSuccess: true, Grade: "100%"}
	mockState.EXPECT().GetSessionResult().Return(expectedResult, nil)

	session.ChangeState(mockState)

	result, err := session.GetSessionResult()

	require.NoError(t, err)
	require.Equal(t, expectedResult, result)
}

func TestSession_GetSessionDurationLimit(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	userID := uint64(1)
	topics := []string{"topic1"}
	generator := testdata.NewMockIDGenerator(ctrl)
	generator.EXPECT().GenerateID().Return(uint64(1))
	storage := testdata.NewMockSessionStorage(ctrl)

	session, err := entities.NewSession(userID, topics, generator, storage)
	require.NoError(t, err)

	mockState := testdata.NewMockSessionState(ctrl)
	expectedDuration := time.Minute * 5
	mockState.EXPECT().GetSessionDurationLimit().Return(expectedDuration, nil)

	session.ChangeState(mockState)

	result, err := session.GetSessionDurationLimit()

	require.NoError(t, err)
	require.Equal(t, expectedDuration, result)
}

func TestSession_IsExpired(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	userID := uint64(1)
	topics := []string{"topic1"}
	generator := testdata.NewMockIDGenerator(ctrl)
	generator.EXPECT().GenerateID().Return(uint64(1))
	storage := testdata.NewMockSessionStorage(ctrl)

	session, err := entities.NewSession(userID, topics, generator, storage)
	require.NoError(t, err)

	mockState := testdata.NewMockSessionState(ctrl)
	mockState.EXPECT().IsExpired().Return(true, nil)

	session.ChangeState(mockState)

	result, err := session.IsExpired()

	require.NoError(t, err)
	require.True(t, result)
}

func TestSession_GetQuestions(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	userID := uint64(1)
	topics := []string{"topic1"}
	generator := testdata.NewMockIDGenerator(ctrl)
	generator.EXPECT().GenerateID().Return(uint64(1))
	storage := testdata.NewMockSessionStorage(ctrl)

	session, err := entities.NewSession(userID, topics, generator, storage)
	require.NoError(t, err)

	mockState := testdata.NewMockSessionState(ctrl)
	mockQuestion := testdata.NewMockQuestion(ctrl)
	expectedQuestions := []entities.Question{mockQuestion}
	mockState.EXPECT().GetQuestions().Return(expectedQuestions, nil)

	session.ChangeState(mockState)

	result, err := session.GetQuestions()

	require.NoError(t, err)
	require.Equal(t, expectedQuestions, result)
}

func TestSession_GetStartedAt(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	userID := uint64(1)
	topics := []string{"topic1"}
	generator := testdata.NewMockIDGenerator(ctrl)
	generator.EXPECT().GenerateID().Return(uint64(1))
	storage := testdata.NewMockSessionStorage(ctrl)

	session, err := entities.NewSession(userID, topics, generator, storage)
	require.NoError(t, err)

	mockState := testdata.NewMockSessionState(ctrl)
	expectedTime := time.Now()
	mockState.EXPECT().GetStartedAt().Return(expectedTime, nil)

	session.ChangeState(mockState)

	result, err := session.GetStartedAt()

	require.NoError(t, err)
	require.Equal(t, expectedTime, result)
}

func TestSession_GetUserAnswers(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	userID := uint64(1)
	topics := []string{"topic1"}
	generator := testdata.NewMockIDGenerator(ctrl)
	generator.EXPECT().GenerateID().Return(uint64(1))
	storage := testdata.NewMockSessionStorage(ctrl)

	session, err := entities.NewSession(userID, topics, generator, storage)
	require.NoError(t, err)

	mockState := testdata.NewMockSessionState(ctrl)
	userAnswer, err := entities.NewUserAnswer(1, []string{"answer"})
	require.NoError(t, err)
	expectedAnswers := []*entities.UserAnswer{userAnswer}
	mockState.EXPECT().GetUserAnswers().Return(expectedAnswers, nil)

	session.ChangeState(mockState)

	result, err := session.GetUserAnswers()

	require.NoError(t, err)
	require.Equal(t, expectedAnswers, result)
}

func Test_NewSession_Failed(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer t.Cleanup(func() {
		ctrl.Finish()
	})

	invalidUserID := uint64(0)
	var invalidGenerator entities.IDGenerator
	var invalidStorage entities.SessionStorage

	userID := uint64(1)
	generator := testdata.NewMockIDGenerator(ctrl)
	storage := testdata.NewMockSessionStorage(ctrl)

	s, err := entities.NewSession(invalidUserID, []string{}, generator, storage)
	require.Nil(t, s)
	require.ErrorIs(t, err, entities.ErrInvalidParam)

	s, err = entities.NewSession(userID, []string{}, invalidGenerator, storage)
	require.Nil(t, s)
	require.ErrorIs(t, err, entities.ErrInvalidParam)

	s, err = entities.NewSession(userID, []string{}, generator, invalidStorage)
	require.Nil(t, s)
	require.ErrorIs(t, err, entities.ErrInvalidParam)

	s, err = entities.NewSession(userID, []string{}, generator, storage)
	require.Nil(t, s)
	require.ErrorIs(t, err, entities.ErrInvalidParam)
}
