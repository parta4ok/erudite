package port

import (
	"context"

	"github.com/parta4ok/kvs/auth/internal/entities"
)

//go:generate mockgen -source=command_factory.go -destination=./testdata/command_factory.go -package=testdata
type CommandFactory interface {
	NewIntrospectedCommand(ctx context.Context, jwt string) (entities.Command, error)
	NewSignInCommand(ctx context.Context, userName string, password string) (entities.Command, error)
	NewAddUserCommand(ctx context.Context, user *entities.User) (entities.Command, error)
	NewDeleteUserCommand(ctx context.Context, userID string) (entities.Command, error)
	NewUpdateUserCommand(ctx context.Context, user *entities.User) (entities.Command, error)
	NewGetLinkedUsersCommand(ctx context.Context, userID string) (entities.Command, error)
	NewAddGroupCommand(ctx context.Context, title string, linkedID string) (entities.Command, error)
	NewGetMentorGroupsCommand(ctx context.Context, mentorID string) (entities.Command, error)
}
