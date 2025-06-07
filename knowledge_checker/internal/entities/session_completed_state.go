package entities

import (
	"fmt"

	"github.com/pkg/errors"
)

var (
	_ SessionState = (*CompletedSessionState)(nil)
)

type CompletedSessionState struct {
	session *Session
}

func NewCompletedSessionState(session *Session) *CompletedSessionState {
	return &CompletedSessionState{
		session: session,
	}
}

func (state *CompletedSessionState) GetStatus() string {
	return CompletedState
}

func (state *CompletedSessionState) SetQuestions(_ *Session, _ map[uint64]Question) error {
	return errors.Wrapf(ErrInvalidState, "%s not support `SetQuetions`", state.GetStatus())
}

func (state *CompletedSessionState) SetUserAnswer(session *Session, answers []UserAnswer) error {
	return errors.Wrapf(ErrInvalidState, "%s not support `SetUserAnswer`", state.GetStatus())
}

func (state *CompletedSessionState) SetSessionID(session *Session, sessessionID uint64) error {
	return errors.Wrapf(ErrInvalidState, "%s not support `SetSessionID`", state.GetStatus())
}

func (state *CompletedSessionState) GetSessionResult(session *Session) (*SessionResult, error) {
	var correctUserAnswers float32
	for _, answer := range session.answers {
		if isCorrectAnswer(session.questions[answer.questionID], answer) {
			correctUserAnswers++
		}
	}

	sessionResult := &SessionResult{}
	if correctUserAnswers == 0 {
		return sessionResult, nil
	}

	percent := (correctUserAnswers / float32(len(session.questions))) * 100
	if percent > 70.0 {
		sessionResult.Done = true
	}

	sessionResult.SuccessPercent = fmt.Sprintf("result is: %1f percents", percent)
	return sessionResult, nil
}

func isCorrectAnswer(question Question, answer UserAnswer) bool {
	if len(answer.answer) == 0 {
		return false
	}

	if len(question.CorrectAnswer()) != len(answer.answer) {
		return false
	}

	for _, selection := range answer.answer {
		if _, ok := question.CorrectAnswer()[selection]; !ok {
			return false
		}
	}

	return true
}
