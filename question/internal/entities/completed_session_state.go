package entities

import (
	"context"
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
	questions map[string]Question
	answers   []*UserAnswer
	holder    StateHolder
	startedAt time.Time
	isExpired bool
}

func NewCompletedSessionState(
	questions map[string]Question,
	holder StateHolder,
	answers []*UserAnswer,
	startedAt time.Time,
	isExpired bool,
) *CompletedSessionState {
	return &CompletedSessionState{
		questions: questions,
		holder:    holder,
		answers:   answers,
		startedAt: startedAt,
		isExpired: isExpired,
	}
}

func (state *CompletedSessionState) GetStatus() string {
	return CompletedState
}

func (state *CompletedSessionState) SetQuestions(_ map[string]Question,
	duration time.Duration) error {
	return errors.Wrapf(ErrInvalidState, "%s not support `SetQuestions`", state.GetStatus())
}

func (state *CompletedSessionState) SetUserAnswer(_ []*UserAnswer) error {
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
				"user anwer has invalid question id: %s", userAnswer.questionID)
		}
		if state.isAnswerCorrect(question, userAnswer) {
			countOfCorrectUserAnswers++
		}
	}

	percent := float64(countOfCorrectUserAnswers) / float64(len(state.questions)) * 100
	usersCorrectAnswersPercent := fmt.Sprintf("%.2f percents", percent)

	return &SessionResult{
		IsSuccess: percent >= DefaultBorderResult,
		Grade:     usersCorrectAnswersPercent,
	}, nil
}

func (state *CompletedSessionState) isAnswerCorrect(qusetion Question, answer *UserAnswer) bool {
	return qusetion.IsAnswerCorrect(answer)
}

func (state *CompletedSessionState) GetSessionDurationLimit() (time.Duration, error) {
	return time.Duration(0), errors.Wrapf(
		ErrInvalidState, "%s not support `GetSessionDurationLimit`", state.GetStatus())
}

func (state *CompletedSessionState) IsExpired() (bool, error) {
	return state.isExpired, nil
}

func (state *CompletedSessionState) GetQuestions() ([]Question, error) {
	questionsList := make([]Question, 0, len(state.questions))
	for _, question := range state.questions {
		questionsList = append(questionsList, question)
	}

	return questionsList, nil
}

func (state *CompletedSessionState) GetStartedAt() (time.Time, error) {
	return state.startedAt, nil
}

func (state *CompletedSessionState) GetUserAnswers() ([]*UserAnswer, error) {
	return state.answers, nil
}

func (state *CompletedSessionState) IsDailySessionLimitReached(ctx context.Context, userID string,
	topics []string) (bool, error) {
	return false, errors.Wrapf(
		ErrInvalidState, "%s not support `IsDailySessionLimitReached`", state.GetStatus())
}
