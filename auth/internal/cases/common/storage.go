package common

import (
	"context"

	"github.com/parta4ok/kvs/auth/internal/entities"
)

//go:generate mockgen -source=storage.go -destination=./testdata/storage.go -package=testdata
type Storage interface {
	GetUserByID(ctx context.Context, userID string) (*entities.User, error)
	GetUserByUsername(ctx context.Context, userName string) (*entities.User, error)
	StoreUser(ctx context.Context, user *entities.User) error
	UpdateUser(ctx context.Context, user *entities.User) error
	RemoveUser(ctx context.Context, userID string) error
}
