package cases

import (
	"context"

	"github.com/parta4ok/kvs/reporting/internal/entities"
)

//go:generate mockgen -source=./auth_client.go -destination=./testdata/auth_client.go -package=testdata
type AuthClient interface {
	GetMentorGroups(ctx context.Context, mentorID string) ([]entities.Student, error)
	GetLinkedUsers(ctx context.Context, id string) (*entities.LinkedMentorAndStudent, error)
	GetUserByID(ctx context.Context, id string) (*entities.User, error)
	Introspect(ctx context.Context, jwt string) (*entities.Claims, error)
	GetGroupTitleByID(ctx context.Context, groupID string) (string, error)
}
