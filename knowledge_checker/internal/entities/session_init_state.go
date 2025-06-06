package entities

import (
	"time"

	"github.com/pkg/errors"
)

const (
	DefaultTopicTimeLimit = time.Minute * 10
)

var (
	_ SessionState = (*InitSessionState)(nil)
)

type InitSessionState struct {
	session *Session
}

func NewInitSessionState(topics []string, userID uint64) *InitSessionState {
	session := &Session{
		userID:   userID,
		topics:   topics,
		duration: DefaultTopicTimeLimit * time.Duration(len(topics)),
	}
	return &InitSessionState{
		session: session,
	}
}

func (state *InitSessionState) GetStatus() string {
	return ActiveState
}

func (state *InitSessionState) SetQuestions(session *Session, qestions map[uint64]Question) error {
	session.questions = qestions
	session.state = NewActiveSessionState(session.userID, session.topics,
		session.duration, session.questions)

	return nil
}

func (state *InitSessionState) SetUserAnswer(_ *Session, _ []UserAnswer) error {
	return errors.Wrapf(ErrInvalidState, "%s not support `SetUserAnswer`", state.GetStatus())
}

func (state *InitSessionState) SetSessionID(session *Session, sessessionID uint64) error {
	session.sessionID = sessessionID
	state.session = session
	return nil
}

func (state *InitSessionState) GetSessionResult(session *Session) (*SessionResult, error) {
	return nil, errors.Wrapf(ErrInvalidState, "%s not support `GetSessionResult`", state.GetStatus())
}
