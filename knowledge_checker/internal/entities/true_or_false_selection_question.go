package entities

import (
	"strings"
)

var (
	_ Question = (*TrueOrFalseSelectionQuestion)(nil)
)

type TrueOrFalseSelectionQuestion struct {
	id            uint64
	topic         string
	subject       string
	correctAnswer bool
}

func NewTrueOrFalseSelectionQuestion(id uint64, topic string, subject string,
	correctAnswer bool) *TrueOrFalseSelectionQuestion {
	return &TrueOrFalseSelectionQuestion{
		id:            id,
		topic:         topic,
		subject:       subject,
		correctAnswer: correctAnswer,
	}
}

func (q *TrueOrFalseSelectionQuestion) ID() uint64 {
	return q.id
}

func (q *TrueOrFalseSelectionQuestion) Type() QuestionType {
	return TrueOrFalse
}

func (q *TrueOrFalseSelectionQuestion) Topic() string {
	return q.topic
}

func (q *TrueOrFalseSelectionQuestion) Subject() string {
	return q.subject
}
func (q *TrueOrFalseSelectionQuestion) Variants() []string {
	return []string{"true", "false"}
}

func (q *TrueOrFalseSelectionQuestion) IsAnswerCorrect(ans *UserAnswer) bool {
	if len(ans.answer) != 1 {
		return false
	}
	var userAns bool
	switch strings.ToLower(ans.answer[0]) {
	case "true":
		userAns = true

	case "false":
		userAns = false
	default:
		return false
	}

	return q.correctAnswer == userAns
}
