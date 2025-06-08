package cases

import (
	"context"

	"github.com/pkg/errors"

	"github.com/parta4ok/kvs/knowledge_checker/internal/entities"
)

type SessionService struct {
	storage Storage
}

func NewSessionService(storage Storage) (*SessionService, error) {
	if storage == nil {
		return nil, errors.Wrapf(entities.ErrInvalidParam, "storage not set")
	}

	return &SessionService{
		storage: storage,
	}, nil
}

func CreateSession(ctx context.Context, userID uint64, topics []string) (*entities.Session, error) {
	return nil, nil
}
