package entities

import (
	"context"
	"time"
)

const (
	InitState      = "init state"
	ActiveState    = "active state"
	CompletedState = "completed state"
)

//go:generate mockgen -source=session_state.go -destination=./testdata/session_state.go -package=testdata
type SessionState interface {
	GetStatus() string
	GetQuestions() ([]Question, error)
	GetStartedAt() (time.Time, error)
	GetUserAnswers() ([]*UserAnswer, error)
	SetQuestions(qestions map[string]Question, duration time.Duration) error
	SetUserAnswer(answers []*UserAnswer) error
	GetSessionResult() (*SessionResult, error)
	GetSessionDurationLimit() (time.Duration, error)
	IsExpired() (bool, error)
	IsDailySessionLimitReached(ctx context.Context, userID string, topics []string) (bool, error)
}

type StateHolder interface {
	ChangeState(state SessionState)
}
