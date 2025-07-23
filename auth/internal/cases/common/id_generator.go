package common

import "context"

//go:generate mockgen -source=id_generator.go -destination=./testdata/id_generator.go -package=testdata

type IDGenerator interface {
	Generate(ctx context.Context) (string, error)
}
