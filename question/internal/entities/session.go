package entities

import (
	"context"
	"time"

	"github.com/pkg/errors"
)

type Session struct {
	userID    string
	sessionID string
	topics    []string

	state SessionState
}

type SessionOption func(*Session)

func WithSessionID(sessionID string) SessionOption {
	return func(s *Session) {
		s.sessionID = sessionID
	}
}

func WithNilState() SessionOption {
	return func(s *Session) {
		s.state = nil
	}
}

func (s *Session) setOptions(opts ...SessionOption) {
	for _, opt := range opts {
		opt(s)
	}
}

func NewSession(userID string, topics []string, generator IDGenerator,
	sessionStorage SessionStorage, opts ...SessionOption) (*Session, error) {
	if userID == "" {
		return nil, errors.Wrap(ErrInvalidParam, "invalid userID")
	}

	if generator == nil {
		return nil, errors.Wrap(ErrInvalidParam, "id generator not set")
	}

	if sessionStorage == nil{
		return nil, errors.Wrap(ErrInvalidParam, "session storage not set")
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

	state := NewInitSessionState(session, sessionStorage)
	session.ChangeState(state)

	session.setOptions(opts...)

	return session, nil
}

func NewSessionWithCustomState(sessionID string, userID string, topics []string,
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

func (s *Session) GetSesionID() string {
	return s.sessionID
}

func (s *Session) GetUserID() string {
	return s.userID
}
func (s *Session) GetTopics() []string {
	return s.topics
}

func (s *Session) ChangeState(state SessionState) {
	s.state = nil
	s.state = state
}

func (s *Session) SetQuestions(qestions map[string]Question, duration time.Duration) error {
	return s.state.SetQuestions(qestions, duration)
}

func (s *Session) SetUserAnswer(answers []*UserAnswer) error {
	return s.state.SetUserAnswer(answers)
}

func (s *Session) GetStatus() string {
	return s.state.GetStatus()
}

func (s *Session) GetSessionResult() (*SessionResult, error) {
	return s.state.GetSessionResult()
}

func (s *Session) GetSessionDurationLimit() (time.Duration, error) {
	return s.state.GetSessionDurationLimit()
}

func (s *Session) IsExpired() (bool, error) {
	return s.state.IsExpired()
}

func (s *Session) GetQuestions() ([]Question, error) {
	return s.state.GetQuestions()
}

func (s *Session) GetStartedAt() (time.Time, error) {
	return s.state.GetStartedAt()
}

func (s *Session) GetUserAnswers() ([]*UserAnswer, error) {
	return s.state.GetUserAnswers()
}

func (s *Session) IsDailySessionLimitReached(ctx context.Context, userID string,
	topics []string) (bool, error) {
	return s.state.IsDailySessionLimitReached(ctx, userID, topics)
}
