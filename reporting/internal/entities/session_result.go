package entities

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"
)

const (
	SessionResultType = "session_result"
)

type SessionResult struct {
	kind       MessageType
	UserID     string
	UserName   string
	GroupTitle string
	Topics     []string
	Questions  map[string][]string
	UserAnswer map[string][]string
	IsExpire   bool
	IsSuccess  bool
	Resume     string
}

func NewSessionResult(
	userID string,
	userName string,
	groupTitle string,
	topics []string,
	questions map[string][]string,
	answers map[string][]string,
	isExpire, isSuccess bool,
	resume string,
) (*SessionResult, error) {
	if strings.TrimSpace(userID) == "" {
		return nil, errors.Wrap(ErrInvalidParam, "user id is empty")
	}

	if strings.TrimSpace(userName) == "" {
		return nil, errors.Wrap(ErrInvalidParam, "user name id is empty")
	}

	if strings.TrimSpace(groupTitle) == "" {
		return nil, errors.Wrap(ErrInvalidParam, "group title id is empty")
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
		UserID:     strings.TrimSpace(userID),
		UserName:   userName,
		GroupTitle: groupTitle,
		Topics:     topics,
		Questions:  questions,
		UserAnswer: answers,
		IsExpire:   isExpire,
		IsSuccess:  isSuccess,
		Resume:     strings.TrimSpace(resume),
	}, nil
}

func (sr *SessionResult) GetReport() interface{} {
	return sr
}

func (sr *SessionResult) Kind() MessageType {
	if sr.kind == MessageType("") {
		return NotificationAboutFinishedSession
	}

	return sr.kind
}

func (sr *SessionResult) SetMessageType() {
	sr.kind = MessageType(fmt.Sprintf("%s.%s_%s",
		ReportType,
		SessionResultType,
		sr.tuneFullname(sr.UserName),
	))
}

func (sr *SessionResult) tuneFullname(fullname string) string {
	splitted := strings.Split(fullname, " ")
	return strings.Join(splitted, "_")
}
