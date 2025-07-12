package jwtprovider

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/parta4ok/kvs/auth/internal/cases/common"
	"github.com/parta4ok/kvs/auth/internal/entities"
	"github.com/pkg/errors"
)

var (
	_ common.JWTProvider = (*Provider)(nil)
)

type Provider struct {
	secret              []byte
	aud                 []string
	iss                 string
	tokenValidityPeriod time.Duration
}

func NewProvider(secret []byte, aud []string, iss string, ttl time.Duration) (*Provider, error) {
	if len(secret) == 0 {
		return nil, errors.Wrap(entities.ErrInvalidParam, "secret not set")
	}

	if iss == "" {
		return nil, errors.Wrap(entities.ErrInvalidParam, "iss not set")
	}

	if ttl == time.Duration(0) {
		return nil, errors.Wrap(entities.ErrInvalidParam, "jwt ttl not set")
	}

	return &Provider{
		secret:              secret,
		aud:                 aud,
		iss:                 iss,
		tokenValidityPeriod: ttl,
	}, nil
}

type UserClaimsDTO struct {
	Username string   `json:"user_name"`
	Subject  uint64   `json:"sub"`
	Rights   []string `json:"rights"`
	jwt.RegisteredClaims
}

func (p *Provider) Generate(user *entities.User) (string, error) {
	if user == nil {
		return "", errors.Wrap(entities.ErrInvalidParam, "user is nil")
	}

	slog.Info("JWT generate started")

	now := time.Now().UTC()
	claims := UserClaimsDTO{
		Username: user.Username,
		Subject:  user.ID,
		Rights:   user.Rights,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    p.iss,
			Audience:  p.aud,
			Subject:   fmt.Sprintf("%d", user.ID),
			ExpiresAt: jwt.NewNumericDate(now.Add(p.tokenValidityPeriod)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString(p.secret)
	if err != nil {
		err = errors.Wrapf(entities.ErrInvalidJWT, "signed of jwt failure: %v", err)
		slog.Error(err.Error())
		return "", err
	}

	slog.Info("JWT generate completed")
	return tokenStr, nil
}

func (p *Provider) Introspect(tokenString string) (*entities.UserClaims, error) {
	slog.Info("Introspect started")

	token, err := jwt.ParseWithClaims(tokenString, &UserClaimsDTO{},
		func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				err := errors.Wrapf(entities.ErrInvalidJWT, "unexpected signing method: %v",
					token.Header["alg"])
				slog.Error(err.Error())
				return nil, err
			}
			return p.secret, nil
		})

	if err != nil {
		err = errors.Wrapf(entities.ErrInvalidJWT, "jwt parse failure: %v", err)
		slog.Error(err.Error())
		return nil, err
	}

	if !token.Valid {
		err = errors.Wrapf(entities.ErrInvalidJWT, "jwt is invalid")
		slog.Error(err.Error())
		return nil, err
	}

	claims, ok := token.Claims.(*UserClaimsDTO)
	if !ok {
		err = errors.Wrapf(entities.ErrInvalidJWT, "extract claims failure")
		slog.Error(err.Error())
		return nil, err
	}

	slog.Info("Introspect completed")
	return &entities.UserClaims{
		Subject:  claims.Subject,
		Username: claims.Username,
		Rights:   claims.Rights,
		Issuer:   claims.Issuer,
		Audience: claims.Audience,
	}, nil
}
