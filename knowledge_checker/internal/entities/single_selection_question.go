package entities

import "github.com/pkg/errors"

var (
	_ Question = (*SingleSelectionQuestion)(nil)
)

type SingleSelectionQuestion struct {
	id           uint64
	questionType QuestionType
	topic        string
	payload      interface{}
	answer       string
}

func NewSingleSelectionQuestion(
	id uint64,
	topic string,
	payload interface{},
	answer string,
) (*SingleSelectionQuestion, error) {
	q := &SingleSelectionQuestion{
		questionType: SingleSelection,
	}

	if id == 0 {
		return nil, errors.Wrap(ErrInvalidParam, "invalid id value")
	}

	if topic == "" {
		return nil, errors.Wrap(ErrInvalidParam, "invalid topic value")
	}

	if answer == "" {
		return nil, errors.Wrap(ErrInvalidParam, "invalid answer value")
	}

	q.id = id
	q.topic = topic
	q.payload = payload
	q.answer = answer

	return q, nil
}

func (q *SingleSelectionQuestion) ID() uint64 {
	return q.id
}

func (q *SingleSelectionQuestion) Type() QuestionType {
	return q.questionType
}

func (q *SingleSelectionQuestion) Topic() string {
	return q.topic
}

func (q *SingleSelectionQuestion) Payload() interface{} {
	return q.payload
}

func (q *SingleSelectionQuestion) CorrectAnswer() map[string]struct{} {
	ans := make(map[string]struct{}, 1)
	ans[q.answer] = struct{}{}
	return ans
}
