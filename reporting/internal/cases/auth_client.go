package cases

import (
	"context"

	"github.com/parta4ok/kvs/reporting/internal/entities"
)

//go:generate mockgen -source=./auth_client.go -destination=./testdata/auth_client.go -package=testdata
type AuthClient interface {
	GetMentorGroups(ctx context.Context, mentorID string) ([]entities.Student, error)
}
