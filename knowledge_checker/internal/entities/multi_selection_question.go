package entities

import "github.com/pkg/errors"

var (
	_ Question = (*MultiSelectionQuestion)(nil)
)

type MultiSelectionQuestion struct {
	id           uint64
	questionType QuestionType
	topic        string
	payload      interface{}
	answer       map[string]struct{}
}

func NewMultiSelectionQuestion(
	id uint64,
	topic string,
	payload interface{},
	answers []string,
) (*MultiSelectionQuestion, error) {
	q := &MultiSelectionQuestion{
		questionType: MultiSelection,
	}

	if id == 0 {
		return nil, errors.Wrap(ErrInvalidParam, "invalid id value")
	}

	if topic == "" {
		return nil, errors.Wrap(ErrInvalidParam, "invalid topic value")
	}

	if len(answers) <= 1 {
		return nil, errors.Wrap(ErrInvalidParam, "invalid answer value")
	}

	q.id = id
	q.topic = topic
	q.payload = payload
	ans := make(map[string]struct{}, len(answers))
	for _, answer := range answers {
		ans[answer] = struct{}{}
	}
	q.answer = ans

	return q, nil
}

func (q *MultiSelectionQuestion) ID() uint64 {
	return q.id
}

func (q *MultiSelectionQuestion) Type() QuestionType {
	return q.questionType
}

func (q *MultiSelectionQuestion) Topic() string {
	return q.topic
}

func (q *MultiSelectionQuestion) Payload() interface{} {
	return q.payload
}

func (q *MultiSelectionQuestion) CorrectAnswer() map[string]struct{} {
	return q.answer
}
