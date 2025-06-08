package entities

import (
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
	SetQuestions(qestions map[uint64]Question, duration time.Duration) error
	SetUserAnswer(answers []UserAnswer) error
	GetSessionResult() (*SessionResult, error)
}

type StateHolder interface {
	ChangeState(state SessionState)
}
