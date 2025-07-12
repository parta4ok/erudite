package entities_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/parta4ok/kvs/question/internal/entities"
)

func TestNewMultiSelectionQuestion(t *testing.T) {
	t.Parallel()

	id := uint64(time.Now().UnixNano())
	topic := "test topic"
	subject := "who is your class mate?"
	variants := []string{"John Doe", "Jack Spearrow", "Jonn Leanon", "John Travolta"}
	correctAnswer := []string{"John Travolta", "Jack Spearrow"}

	question := entities.NewMultiSelectionQuestion(id, topic, subject, variants, correctAnswer)
	require.NotNil(t, question)

	notCorrectAns, err := entities.NewUserAnswer(question.ID(),
		[]string{"Jonn Leanon", "Jack Spearrow"})
	require.NoError(t, err)

	correctAns, err := entities.NewUserAnswer(question.ID(),
		[]string{"John Travolta", "Jack Spearrow"})
	require.NoError(t, err)

	res := question.IsAnswerCorrect(notCorrectAns)
	require.False(t, res)

	res = question.IsAnswerCorrect(correctAns)
	require.True(t, res)

	require.Equal(t, id, question.ID())
	require.Equal(t, topic, question.Topic())
	require.Equal(t, subject, question.Subject())
	require.Equal(t, variants, question.Variants())
	require.Equal(t, entities.MultiSelection, question.Type())
}

func TestMultiSelectionQuestion_IsAnswerCorrect_EmptyAnswer(t *testing.T) {
	t.Parallel()

	question := entities.NewMultiSelectionQuestion(1, "topic", "subject",
		[]string{"A", "B", "C"}, []string{"A", "B"})

	emptyAnswer, err := entities.NewUserAnswer(question.ID(), []string{})
	require.NoError(t, err)

	result := question.IsAnswerCorrect(emptyAnswer)

	require.False(t, result)
}

func TestMultiSelectionQuestion_IsAnswerCorrect_PartialCorrect(t *testing.T) {
	t.Parallel()

	question := entities.NewMultiSelectionQuestion(1, "topic", "subject",
		[]string{"A", "B", "C", "D"}, []string{"A", "B", "C"})

	partialAnswer, err := entities.NewUserAnswer(question.ID(), []string{"A", "B"})
	require.NoError(t, err)

	result := question.IsAnswerCorrect(partialAnswer)

	require.False(t, result)
}

func TestMultiSelectionQuestion_IsAnswerCorrect_ExtraAnswers(t *testing.T) {
	t.Parallel()

	question := entities.NewMultiSelectionQuestion(1, "topic", "subject",
		[]string{"A", "B", "C", "D"}, []string{"A", "B"})

	extraAnswer, err := entities.NewUserAnswer(question.ID(), []string{"A", "B", "C"})
	require.NoError(t, err)

	result := question.IsAnswerCorrect(extraAnswer)

	require.False(t, result)
}
