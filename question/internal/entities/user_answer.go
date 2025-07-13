package entities

import "github.com/pkg/errors"

type UserAnswer struct {
	questionID string
	answer     []string
}

func NewUserAnswer(id string, answer []string) (*UserAnswer, error) {
	if id == "" {
		return nil, errors.Wrap(ErrUnprocessibleEntity, "invalid id")
	}

	return &UserAnswer{
		questionID: id,
		answer:     answer,
	}, nil
}

func (ans *UserAnswer) GetQuestionID() string {
	return ans.questionID
}

func (ans *UserAnswer) GetSelections() []string {
	return ans.answer
}
