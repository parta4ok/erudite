package public

import (
	"context"

	"github.com/parta4ok/kvs/knowledge_checker/internal/entities"
)

//go:generate mockgen -source=service.go -destination=./testdata/service.go -package=testdata
type Service interface {
	CompleteSession(ctx context.Context, sessionID uint64, answers []entities.UserAnswer) (
		*entities.SessionResult, error)
	CreateSession(ctx context.Context, userID uint64, topics []string) (
		uint64, map[uint64]entities.Question, error)
	ShowTopics(ctx context.Context) ([]string, error)
}
