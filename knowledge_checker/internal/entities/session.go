package entities

import (
	"time"
)

const (
	InitState      = "init state"
	ActiveState    = "active state"
	CompletedState = "completed state"
	ExpiredState   = "expired state"
)

type SessionState interface {
	SetSessionID(session *Session, sessionID uint64) error
	SetQuestions(session *Session, qestions map[uint64]Question) error
	SetUserAnswer(session *Session, answers []UserAnswer) error
	GetStatus() string
	GetSessionResult(session *Session) (*SessionResult, error)
}

type Session struct {
	sessionID uint64
	userID    uint64
	topics    []string
	state     SessionState
	startedAt time.Time
	duration  time.Duration
	questions map[uint64]Question
	answers   []UserAnswer
}

type SessionResult struct {
	Done           bool
	SuccessPercent string
}
