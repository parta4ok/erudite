package port

import (
	"context"

	"github.com/parta4ok/kvs/auth/internal/entities"
)

//go:generate mockgen -source=command_factory.go -destination=./testdata/command_factory.go -package=testdata
type CommandFactory interface {
	NewIntrospectedCommand(ctx context.Context, jwt string) (entities.Command, error)
	NewSignInCommand(ctx context.Context, userName string, password string) (entities.Command, error)
	NewAddUserCommand(ctx context.Context, login string, password string, rights []string,
		contacts map[string]string) (entities.Command, error)
	NewDeleteUserCommand(ctx context.Context, userID string) (entities.Command, error)
}
