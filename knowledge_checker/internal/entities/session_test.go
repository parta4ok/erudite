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
