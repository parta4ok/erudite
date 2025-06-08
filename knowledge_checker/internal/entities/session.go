package entities

import (
	"time"

	"github.com/pkg/errors"
)

type Session struct {
	userID    uint64
	sessionID uint64
	topics    []string

	state SessionState
}

func NewSession(userID uint64, topics []string, generator IDGenerator) (*Session, error) {
	if userID == 0 {
		return nil, errors.Wrap(ErrInvalidParam, "invalid userID")
	}

	if generator == nil {
		return nil, errors.Wrap(ErrInvalidParam, "id generator not set")
	}

	if len(topics) == 0 {
		return nil, errors.Wrap(ErrInvalidParam, "topics was not selected")
	}

	sessionID := generator.GenerateID()

	session := &Session{
		userID:    userID,
		sessionID: sessionID,
		topics:    topics,
	}

	state := NewInitSessionState(session)
	session.ChangeState(state)

	return session, nil
}

func NewSessionWithCustomState(userID uint64, topics []string, sessionID uint64,
	state SessionState) *Session {
	return &Session{
		userID:    userID,
		sessionID: sessionID,
		topics:    topics,
		state:     state,
	}
}

type SessionResult struct {
	IsSuccess bool
	Grade     string
}

func (s *Session) GetSesionID() uint64 {
	return s.sessionID
}

func (s *Session) ChangeState(state SessionState) {
	s.state = state
}

func (s *Session) SetQuestions(qestions map[uint64]Question, duration time.Duration) error {
	return s.state.SetQuestions(qestions, duration)
}

func (s *Session) SetUserAnswer(answers []UserAnswer) error {
	return s.state.SetUserAnswer(answers)
}

func (s *Session) GetStatus() string {
	return s.state.GetStatus()
}

func (s *Session) GetSessionResult() (*SessionResult, error) {
	return s.state.GetSessionResult()
}
