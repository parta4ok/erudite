package private

import (
	"context"

	"github.com/parta4ok/kvs/auth/internal/entities"
)

type CommandFactory interface {
	NewIntrospectedCommand(ctx context.Context, userID uint64, jwt string) (entities.Command, error)
	NewSignInCommand(ctx context.Context, userName string, password string) (entities.Command, error)
}
