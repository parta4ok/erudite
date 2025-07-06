package common

import "github.com/parta4ok/kvs/auth/internal/entities"

//go:generate mockgen -source=jwt_provider.go -destination=./testdata/jwt_provider.go -package=testdata
type JWTProvider interface {
	Generate(userClaims *entities.User) (string, error)
	Introspect(token string) (*entities.UserClaims, error)
}
