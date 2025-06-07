package entities

import (
	"time"

	"github.com/pkg/errors"
)

const (
	InitState      = "init state"
	ActiveState    = "active state"
	CompletedState = "completed state"
	ExpiredState   = "expired state"
)

//go:generate mockgen -source=session.go -destination=./testdata/session.go -package=testdata
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

func (s *Session) SetSessionID(session *Session, sessionID uint64) error {
	if session == nil {
		return errors.Wrap(ErrInvalidParam, "session is nil")
	}

	return s.state.SetSessionID(session, sessionID)
}

func (s *Session) SetQuestions(session *Session, qestions map[uint64]Question) error {
	if session == nil {
		return errors.Wrap(ErrInvalidParam, "session is nil")
	}

	return s.state.SetQuestions(session, qestions)
}

func (s *Session) SetUserAnswer(session *Session, answers []UserAnswer) error {
	if session == nil {
		return errors.Wrap(ErrInvalidParam, "session is nil")
	}

	return s.state.SetUserAnswer(session, answers)
}

func (s *Session) GetStatus() string {
	return s.state.GetStatus()
}

func (s *Session) GetSessionResult(session *Session) (*SessionResult, error) {
	if session == nil {
		return nil, errors.Wrap(ErrInvalidParam, "session is nil")
	}

	return s.state.GetSessionResult(session)
}
