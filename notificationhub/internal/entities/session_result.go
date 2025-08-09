package entities

import (
	"strings"

	"github.com/pkg/errors"
)

type SessionResult struct {
	userID     string
	Topics     []string
	Questions  map[int]string
	UserAnswer map[string][]string
	IsExpire   bool
	IsSuccess  bool
	Resume     string
}

func NewSessionResult(
	userID string,
	topics []string,
	questions map[int]string,
	answers map[string][]string,
	isExpire, isSuccess bool,
	resume string,
) (*SessionResult, error) {
	if strings.TrimSpace(userID) == "" {
		return nil, errors.Wrap(ErrInvalidParam, "user id is empty")
	}

	if len(topics) == 0 {
		return nil, errors.Wrap(ErrInvalidParam, "topics list is empty")
	}

	if len(questions) == 0 {
		return nil, errors.Wrap(ErrInvalidParam, "questions list is empty")
	}

	if len(answers) == 0 {
		return nil, errors.Wrap(ErrInvalidParam, "answers list is empty")
	}

	if strings.TrimSpace(resume) == "" {
		return nil, errors.Wrap(ErrInvalidParam, "resume is empty")
	}

	return &SessionResult{
		userID:     strings.TrimSpace(userID),
		Topics:     topics,
		Questions:  questions,
		UserAnswer: answers,
		IsExpire:   isExpire,
		IsSuccess:  isSuccess,
		Resume:     strings.TrimSpace(resume),
	}, nil
}

func (sr *SessionResult) GetUserID() string {
	return sr.userID
}
