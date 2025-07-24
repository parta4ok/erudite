package public

import "context"

//go:generate mockgen -source=accessor.go -destination=./testdata/accessor.go -package=testdata
type Accessor interface {
	HasPermission(ctx context.Context, rights []string) (bool, error)
}
