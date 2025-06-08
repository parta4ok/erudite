package entities

import (
	"fmt"
	"time"

	"github.com/pkg/errors"
)

var (
	_ SessionState = (*CompletedSessionState)(nil)
)

const (
	DefaultBorderResult = 60.0
)

type CompletedSessionState struct {
	questions map[uint64]Question
	answers   []UserAnswer
	holder    StateHolder
	isExpired bool
}

func NewCompletedSessionState(
	questions map[uint64]Question,
	holder StateHolder,
	answers []UserAnswer,
	isExpired bool,
) *CompletedSessionState {
	return &CompletedSessionState{
		questions: questions,
		holder:    holder,
		answers:   answers,
		isExpired: isExpired,
	}
}

func (state *CompletedSessionState) GetStatus() string {
	return CompletedState
}

func (state *CompletedSessionState) SetQuestions(qestions map[uint64]Question, duration time.Duration) error {
	return errors.Wrapf(ErrInvalidState, "%s not support `SetQuestions`", state.GetStatus())
}

func (state *CompletedSessionState) SetUserAnswer(answers []UserAnswer) error {
	return errors.Wrapf(ErrInvalidState, "%s not support `SetUserAnswer`", state.GetStatus())
}

func (state *CompletedSessionState) GetSessionResult() (*SessionResult, error) {
	if state.isExpired {
		return &SessionResult{
			IsSuccess: false,
			Grade:     "session expired",
		}, nil
	}

	return state.processingResult()
}

func (state *CompletedSessionState) processingResult() (*SessionResult, error) {
	var countOfCorrectUserAnswers int

	for _, userAnswer := range state.answers {
		question, ok := state.questions[userAnswer.questionID]
		if !ok {
			return nil, errors.Wrapf(ErrInvalidParam,
				"user anwer has invalid question id: %d", userAnswer.questionID)
		}
		if state.isAnswerCorrect(question, userAnswer) {
			countOfCorrectUserAnswers++
		}
	}

	percent := float64(countOfCorrectUserAnswers) / float64(len(state.questions))
	usersCorrectAnswersPercent := fmt.Sprintf("%.2f percents", percent)

	return &SessionResult{
		IsSuccess: percent >= DefaultBorderResult,
		Grade:     usersCorrectAnswersPercent,
	}, nil
}

func (state *CompletedSessionState) isAnswerCorrect(qusetion Question, answer UserAnswer) bool {
	correctAnswer := qusetion.CorrectAnswer()
	userAnswer := answer.answer

	if len(correctAnswer) != len(userAnswer) {
		return false
	}

	for _, selection := range userAnswer {
		if _, ok := correctAnswer[selection]; !ok {
			return false
		}
	}

	return true
}
