package cases

import (
	"context"

	"github.com/parta4ok/kvs/knowledge_checker/internal/entities"
)

//go:generate mockgen -source=storage.go -destination=./testdata/storage.go -package=testdata
type Storage interface {
	GetTopics(ctx context.Context) ([]string, error)
	GetQuesions(ctx context.Context, topics []string) ([]entities.Question, error)
	StoreSession(ctx context.Context, session *entities.Session) error
	GetSessionBySessionID(ctx context.Context, sessionID uint64) (*entities.Session, error)
}
