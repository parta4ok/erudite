package entities_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/parta4ok/kvs/question/internal/entities"
)

func TestQuestionType_String(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name         string
		questionType entities.QuestionType
		expected     string
	}{
		{
			name:         "single_selection",
			questionType: entities.SingleSelection,
			expected:     "single selection",
		},
		{
			name:         "multi_selection",
			questionType: entities.MultiSelection,
			expected:     "multi selection",
		},
		{
			name:         "true_or_false",
			questionType: entities.TrueOrFalse,
			expected:     "true or false",
		},
		{
			name:         "unknown_type",
			questionType: entities.QuestionType(999),
			expected:     "",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := tc.questionType.String()

			require.Equal(t, tc.expected, result)
		})
	}
}

func TestQuestionFactory_NewQuestion_SingleSelection_Success(t *testing.T) {
	t.Parallel()

	factory := &entities.QuestionFactory{}
	id := uint64(1)
	topic := "Go"
	subject := "What is Go?"
	variants := []string{"Language", "Game", "Tool", "Framework"}
	correctAnswer := []string{"Language"}

	question, err := factory.NewQuestion(id, entities.SingleSelection, topic, subject,
		variants, correctAnswer)

	require.NoError(t, err)
	require.NotNil(t, question)
	require.Equal(t, id, question.ID())
	require.Equal(t, entities.SingleSelection, question.Type())
	require.Equal(t, topic, question.Topic())
	require.Equal(t, subject, question.Subject())
	require.Equal(t, variants, question.Variants())
}

func TestQuestionFactory_NewQuestion_MultiSelection_Success(t *testing.T) {
	t.Parallel()

	factory := &entities.QuestionFactory{}
	id := uint64(2)
	topic := "Programming"
	subject := "Which are programming languages?"
	variants := []string{"Go", "Python", "HTML", "JavaScript"}
	correctAnswer := []string{"Go", "Python", "JavaScript"}

	question, err := factory.NewQuestion(id, entities.MultiSelection, topic, subject, variants,
		correctAnswer)

	require.NoError(t, err)
	require.NotNil(t, question)
	require.Equal(t, id, question.ID())
	require.Equal(t, entities.MultiSelection, question.Type())
}

func TestQuestionFactory_NewQuestion_TrueOrFalse_Success(t *testing.T) {
	t.Parallel()

	factory := &entities.QuestionFactory{}
	id := uint64(3)
	topic := "Go"
	subject := "Go is compiled language"
	correctAnswer := []string{"true"}

	question, err := factory.NewQuestion(id, entities.TrueOrFalse, topic, subject, nil, correctAnswer)

	require.NoError(t, err)
	require.NotNil(t, question)
	require.Equal(t, id, question.ID())
	require.Equal(t, entities.TrueOrFalse, question.Type())
}

func TestQuestionFactory_NewQuestion_TrueOrFalse_False(t *testing.T) {
	t.Parallel()

	factory := &entities.QuestionFactory{}
	id := uint64(4)
	topic := "Go"
	subject := "Go is interpreted language"
	correctAnswer := []string{"false"}

	question, err := factory.NewQuestion(id, entities.TrueOrFalse, topic, subject, nil, correctAnswer)

	require.NoError(t, err)
	require.NotNil(t, question)
}

func TestQuestionFactory_NewQuestion_InvalidID(t *testing.T) {
	t.Parallel()

	factory := &entities.QuestionFactory{}
	id := uint64(0)
	topic := "Go"
	subject := "What is Go?"
	variants := []string{"Language"}
	correctAnswer := []string{"Language"}

	question, err := factory.NewQuestion(id, entities.SingleSelection, topic, subject, variants,
		correctAnswer)

	require.Error(t, err)
	require.Nil(t, question)
	require.Contains(t, err.Error(), "id is 0")
}

func TestQuestionFactory_NewQuestion_EmptyTopic(t *testing.T) {
	t.Parallel()

	factory := &entities.QuestionFactory{}
	id := uint64(1)
	topic := ""
	subject := "What is Go?"
	variants := []string{"Language"}
	correctAnswer := []string{"Language"}

	question, err := factory.NewQuestion(id, entities.SingleSelection, topic, subject,
		variants, correctAnswer)

	require.Error(t, err)
	require.Nil(t, question)
	require.Contains(t, err.Error(), "topic is empty")
}

func TestQuestionFactory_NewQuestion_EmptySubject(t *testing.T) {
	t.Parallel()

	factory := &entities.QuestionFactory{}
	id := uint64(1)
	topic := "Go"
	subject := ""
	variants := []string{"Language"}
	correctAnswer := []string{"Language"}

	question, err := factory.NewQuestion(id, entities.SingleSelection, topic, subject, variants,
		correctAnswer)

	require.Error(t, err)
	require.Nil(t, question)
	require.Contains(t, err.Error(), "subject is empty")
}

func TestQuestionFactory_NewQuestion_SingleSelection_TooManyVariants(t *testing.T) {
	t.Parallel()

	factory := &entities.QuestionFactory{}
	id := uint64(1)
	topic := "Go"
	subject := "What is Go?"
	variants := []string{"A", "B", "C", "D", "E"}
	correctAnswer := []string{"A"}

	question, err := factory.NewQuestion(id, entities.SingleSelection, topic, subject,
		variants, correctAnswer)

	require.Error(t, err)
	require.Nil(t, question)
	require.Contains(t, err.Error(), "variants must be equal or greater then lentgh 4")
}

func TestQuestionFactory_NewQuestion_SingleSelection_MultipleCorrectAnswers(t *testing.T) {
	t.Parallel()

	factory := &entities.QuestionFactory{}
	id := uint64(1)
	topic := "Go"
	subject := "What is Go?"
	variants := []string{"A", "B", "C", "D"}
	correctAnswer := []string{"A", "B"}

	question, err := factory.NewQuestion(id, entities.SingleSelection, topic, subject,
		variants, correctAnswer)

	require.Error(t, err)
	require.Nil(t, question)
	require.Contains(t, err.Error(), "only one correct answer for this question type")
}

func TestQuestionFactory_NewQuestion_MultiSelection_NoCorrectAnswers(t *testing.T) {
	t.Parallel()

	factory := &entities.QuestionFactory{}
	id := uint64(1)
	topic := "Go"
	subject := "What is Go?"
	variants := []string{"A", "B", "C", "D"}
	correctAnswer := []string{}

	question, err := factory.NewQuestion(id, entities.MultiSelection, topic, subject,
		variants, correctAnswer)

	require.Error(t, err)
	require.Nil(t, question)
	require.Contains(t, err.Error(), "minimum one correct answer")
}

func TestQuestionFactory_NewQuestion_TrueOrFalse_MultipleCorrectAnswers(t *testing.T) {
	t.Parallel()

	factory := &entities.QuestionFactory{}
	id := uint64(1)
	topic := "Go"
	subject := "What is Go?"
	correctAnswer := []string{"true", "false"}

	question, err := factory.NewQuestion(id, entities.TrueOrFalse, topic, subject, nil, correctAnswer)

	require.Error(t, err)
	require.Nil(t, question)
	require.Contains(t, err.Error(), "only one correct answer for this question type")
}

func TestQuestionFactory_NewQuestion_UnknownType(t *testing.T) {
	t.Parallel()

	factory := &entities.QuestionFactory{}
	id := uint64(1)
	topic := "Go"
	subject := "What is Go?"
	variants := []string{"A"}
	correctAnswer := []string{"A"}

	question, err := factory.NewQuestion(id, entities.QuestionType(999), topic, subject,
		variants, correctAnswer)

	require.Error(t, err)
	require.Nil(t, question)
	require.Contains(t, err.Error(), "unknown question type: 999")
}

func TestQuestionFactory_NewQuestion_InvalidVariants(t *testing.T) {
	t.Parallel()

	factory := &entities.QuestionFactory{}
	id := uint64(1)
	topic := "Go"
	subject := "What is Go?"
	variants := []string{"A", "B", "C", "D", "E"}
	correctAnswer := []string{"A"}

	question, err := factory.NewQuestion(id, entities.MultiSelection, topic, subject,
		variants, correctAnswer)

	require.ErrorIs(t, err, entities.ErrInvalidParam)
	require.Nil(t, question)
}
