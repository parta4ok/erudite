package entities_test

import (
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

	generator := testdata.NewMockIDGenerator(ctrl)
	generator.EXPECT().GenerateID().Return(uint64(2))

	question := testdata.NewMockQuestion(ctrl)

	userID := 1
	topics := []string{"1"}

	session, err := entities.NewSession(uint64(userID), topics, generator)
	require.NoError(t, err)
	require.NotNil(t, session)

	require.Equal(t, entities.InitState, session.GetStatus())

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
