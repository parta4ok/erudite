package entities

import (
	"strings"

	"github.com/pkg/errors"
)

const (
	SingleSelection QuestionType = iota + 1
	MultiSelection
	TrueOrFalse
)

type QuestionType int

func (q QuestionType) String() string {
	switch q {
	case SingleSelection:
		return "single selection"
	case MultiSelection:
		return "multi selection"
	case TrueOrFalse:
		return "true or false"
	}
	return ""
}

//go:generate mockgen -source=question.go -destination=./testdata/question.go -package=testdata
type Question interface {
	ID() string
	Type() QuestionType
	Topic() string
	Subject() string
	Variants() []string
	IsAnswerCorrect(ans *UserAnswer) bool
}

type QuestionFactory struct{}

func (factory *QuestionFactory) NewQuestion(id string, questionType QuestionType,
	topic string, subject string, variants []string, correctAnswer []string) (Question, error) {
	if id == "" {
		return nil, errors.Wrap(ErrInvalidParam, "invalid id")
	}
	if topic == "" {
		return nil, errors.Wrap(ErrInvalidParam, "topic is empty")
	}
	if subject == "" {
		return nil, errors.Wrap(ErrInvalidParam, "subject is empty")
	}

	switch questionType {
	case SingleSelection:
		if len(variants) > 4 {
			return nil, errors.Wrap(ErrInvalidParam,
				"variants must be equal or greater then lentgh 4")
		}
		if len(correctAnswer) != 1 {
			return nil, errors.Wrap(ErrInvalidParam,
				"only one correct answer for this question type")
		}
		return NewSingleSelectionQuestion(id, topic, subject, variants, correctAnswer[0]), nil

	case MultiSelection:
		if len(variants) > 4 {
			return nil, errors.Wrap(ErrInvalidParam,
				"variants must be equal or greater then lentgh 4")
		}
		if len(correctAnswer) < 1 {
			return nil, errors.Wrap(ErrInvalidParam,
				"minimum one correct answer for multi selection question question")
		}
		return NewMultiSelectionQuestion(id, topic, subject, variants, correctAnswer), nil

	case TrueOrFalse:
		if len(correctAnswer) != 1 {
			return nil, errors.Wrap(ErrInvalidParam,
				"only one correct answer for this question type")
		}
		var ca bool
		switch strings.ToLower(correctAnswer[0]) {
		case "true":
			ca = true
		case "false":
			ca = false
		}
		return NewTrueOrFalseSelectionQuestion(id, topic, subject, ca), nil

	}

	return nil, errors.Wrapf(ErrInvalidParam, "unknown question type: %d", questionType)
}
