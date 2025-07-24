package common

import "context"

//go:generate mockgen -source=hasher.go -destination=./testdata/hasher.go -package=testdata
type Hasher interface {
	Hash(ctx context.Context, pass string) (string, error)
	IsHash(ctx context.Context, password string, hash string) (bool, error)
}
