package entities_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/parta4ok/kvs/question/internal/entities"
)

func TestNewTrueOrFalseQuestion(t *testing.T) {
	t.Parallel()

	id := fmt.Sprintf("%d", uint64(time.Now().UnixNano()))
	topic := "test topic"
	subject := "this test writes with Go?"
	variants := []string{"true", "false"}

	question := entities.NewTrueOrFalseSelectionQuestion(id, topic, subject, true)
	require.NotNil(t, question)

	notCorrectAns, err := entities.NewUserAnswer(question.ID(),
		[]string{"False"})
	require.NoError(t, err)

	correctAns, err := entities.NewUserAnswer(question.ID(),
		[]string{"TRUE"})
	require.NoError(t, err)

	res := question.IsAnswerCorrect(notCorrectAns)
	require.False(t, res)

	res = question.IsAnswerCorrect(correctAns)
	require.True(t, res)

	require.Equal(t, id, question.ID())
	require.Equal(t, topic, question.Topic())
	require.Equal(t, subject, question.Subject())
	require.Equal(t, variants, question.Variants())
	require.Equal(t, entities.TrueOrFalse, question.Type())
}

func TestTrueOrFalseQuestion_FalseAnswer(t *testing.T) {
	t.Parallel()

	question := entities.NewTrueOrFalseSelectionQuestion("1", "topic", "subject", false)

	correctAnswer, err := entities.NewUserAnswer(question.ID(), []string{"false"})
	require.NoError(t, err)

	result := question.IsAnswerCorrect(correctAnswer)

	require.True(t, result)
}

func TestTrueOrFalseQuestion_EmptyAnswer(t *testing.T) {
	t.Parallel()

	question := entities.NewTrueOrFalseSelectionQuestion("1", "topic", "subject", true)

	emptyAnswer, err := entities.NewUserAnswer(question.ID(), []string{})
	require.NoError(t, err)

	result := question.IsAnswerCorrect(emptyAnswer)

	require.False(t, result)
}

func TestTrueOrFalseQuestion_MultipleAnswers(t *testing.T) {
	t.Parallel()

	question := entities.NewTrueOrFalseSelectionQuestion("1", "topic", "subject", true)

	multipleAnswers, err := entities.NewUserAnswer(question.ID(), []string{"true", "false"})
	require.NoError(t, err)

	result := question.IsAnswerCorrect(multipleAnswers)

	require.False(t, result)
}
