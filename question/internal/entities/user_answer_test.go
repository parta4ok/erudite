package entities_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/parta4ok/kvs/question/internal/entities"
)

func TestNewUserAnswer_Success(t *testing.T) {
	t.Parallel()

	questionID := "testID"
	answers := []string{"answer1", "answer2"}

	userAnswer, err := entities.NewUserAnswer(questionID, answers)

	require.NoError(t, err)
	require.NotNil(t, userAnswer)
	require.Equal(t, questionID, userAnswer.GetQuestionID())
	require.Equal(t, answers, userAnswer.GetSelections())
}

func TestNewUserAnswer_EmptyAnswers(t *testing.T) {
	t.Parallel()

	questionID := "testID"
	var answers []string

	userAnswer, err := entities.NewUserAnswer(questionID, answers)

	require.NoError(t, err)
	require.NotNil(t, userAnswer)
	require.Equal(t, questionID, userAnswer.GetQuestionID())
	require.Empty(t, userAnswer.GetSelections())
}

func TestNewUserAnswer_InvalidID(t *testing.T) {
	t.Parallel()

	questionID := ""
	answers := []string{"answer1"}

	userAnswer, err := entities.NewUserAnswer(questionID, answers)

	require.Error(t, err)
	require.Nil(t, userAnswer)
	require.Contains(t, err.Error(), "invalid id")
}

func TestUserAnswer_GetQuestionID(t *testing.T) {
	t.Parallel()

	questionID := "testID"
	answers := []string{"test"}

	userAnswer, err := entities.NewUserAnswer(questionID, answers)
	require.NoError(t, err)

	result := userAnswer.GetQuestionID()

	require.Equal(t, questionID, result)
}

func TestUserAnswer_GetSelections(t *testing.T) {
	t.Parallel()

	questionID := "testID"
	answers := []string{"option1", "option2", "option3"}

	userAnswer, err := entities.NewUserAnswer(questionID, answers)
	require.NoError(t, err)

	result := userAnswer.GetSelections()

	require.Equal(t, answers, result)
	require.Len(t, result, 3)
}
