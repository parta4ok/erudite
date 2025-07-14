package public

import (
	"context"

	"github.com/parta4ok/kvs/question/internal/entities"
)

//go:generate mockgen -source=service.go -destination=./testdata/service.go -package=testdata
type Service interface {
	CompleteSession(ctx context.Context, sessionID string, answers []*entities.UserAnswer) (
		*entities.SessionResult, error)
	CreateSession(ctx context.Context, userID string, topics []string) (
		string, map[string]entities.Question, error)
	ShowTopics(ctx context.Context) ([]string, error)
}
