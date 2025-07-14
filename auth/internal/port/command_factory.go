package port

import (
	"context"

	"github.com/parta4ok/kvs/auth/internal/entities"
)

//go:generate mockgen -source=command_factory.go -destination=./testdata/command_factory.go -package=testdata
type CommandFactory interface {
	NewIntrospectedCommand(ctx context.Context, userID string, jwt string) (entities.Command, error)
	NewSignInCommand(ctx context.Context, userName string, password string) (entities.Command, error)
}
