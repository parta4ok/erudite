package cases

import (
	"context"

	"github.com/parta4ok/kvs/question/internal/entities"
)

//go:generate mockgen -source=./session_service.go -destination=./testdata/session_service.go -package=testdata
type SessionService interface {
	CompleteSession(ctx context.Context, sessionID string, answers []*entities.UserAnswer) (
		*entities.SessionResult, error)
	CreateSession(ctx context.Context, userID string, topics []string) (
		string, map[string]entities.Question, error)
	GetAllCompletedUserSessions(ctx context.Context, userID string) ([]*entities.Session, error)
	ShowTopics(ctx context.Context) ([]string, error)
}
