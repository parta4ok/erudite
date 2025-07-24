package public

import (
	"context"

	"github.com/parta4ok/kvs/question/internal/entities"
)

//go:generate mockgen -source=introspector.go -destination=./testdata/introspector.go -package=testdata
type Introspector interface {
	Introspect(ctx context.Context, jwt string) (*entities.Claims, error)
}
