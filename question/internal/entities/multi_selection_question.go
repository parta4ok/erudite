package entities

import (
	"slices"
	"sort"
)

var (
	_ Question = (*MultiSelectionQuestion)(nil)
)

type MultiSelectionQuestion struct {
	id             string
	topic          string
	subject        string
	variants       []string
	correctAnswers []string
}

func NewMultiSelectionQuestion(id string, topic string, subject string, variants []string,
	correctAnswers []string) *MultiSelectionQuestion {
	return &MultiSelectionQuestion{
		id:             id,
		topic:          topic,
		subject:        subject,
		variants:       variants,
		correctAnswers: correctAnswers,
	}
}

func (q *MultiSelectionQuestion) ID() string {
	return q.id
}

func (q *MultiSelectionQuestion) Type() QuestionType {
	return MultiSelection
}

func (q *MultiSelectionQuestion) Topic() string {
	return q.topic
}

func (q *MultiSelectionQuestion) Subject() string {
	return q.subject
}

func (q *MultiSelectionQuestion) Variants() []string {
	return q.variants
}

func (q *MultiSelectionQuestion) IsAnswerCorrect(ans *UserAnswer) bool {
	if len(q.correctAnswers) != len(ans.answer) {
		return false
	}

	correct := sort.StringSlice(q.correctAnswers)
	correct.Sort()

	user := sort.StringSlice(ans.answer)
	user.Sort()

	return slices.Equal(correct, user)
}
