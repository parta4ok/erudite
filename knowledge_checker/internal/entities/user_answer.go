package entities

import "github.com/pkg/errors"

type UserAnswer struct {
	questionID uint64
	answer     []string
}

func NewUserAnswer(id uint64, answer []string) (*UserAnswer, error) {
	if id == 0 {
		return nil, errors.Wrap(ErrUnprocessibleEntity, "invalid id")
	}

	return &UserAnswer{
		questionID: id,
		answer:     answer,
	}, nil
}
