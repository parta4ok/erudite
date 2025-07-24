package authservice

import (
	"context"
	"log/slog"

	authv1 "github.com/parta4ok/kvs/api/grpc/v1"
	"github.com/parta4ok/kvs/question/internal/entities"
	"github.com/parta4ok/kvs/question/internal/port/http/public"
	"github.com/parta4ok/kvs/toolkit/pkg/auth/client"
	"github.com/pkg/errors"
)

var (
	_ public.Introspector = (*AuthService)(nil)
)

type AuthService struct {
	client *client.AuthClient
}

func NewAuthService(port string) (*AuthService, error) {
	if port == "" {
		err := errors.Wrap(entities.ErrInvalidParam, "port not set")
		return nil, err
	}

	c, err := client.New(port)
	if err != nil {
		err = errors.Wrap(entities.ErrInternal, "creating auth grpc client failure")
		return nil, err
	}

	return &AuthService{client: c}, nil
}

func (srv *AuthService) Introspect(ctx context.Context, jwt string) (*entities.Claims, error) {
	slog.Info("Introspect started")

	req := &authv1.IntrospectRequest{
		Token: jwt,
	}

	resp, err := srv.client.Introspect(ctx, req)
	if err != nil {
		err = errors.Wrapf(entities.ErrInternal, "introspect failure: %v", err)
		slog.Error(err.Error())
		return nil, err
	}

	if resp.Error.Message != "" {
		err := errors.Wrapf(entities.ErrForbidden, "error message: %s", resp.Error.Message)
		slog.Error(err.Error())
		return nil, err
	}

	if resp.Claims == nil {
		err := errors.Wrap(entities.ErrForbidden, "nil claims")
		slog.Error(err.Error())
		return nil, err
	}

	slog.Info("Introspect completed")

	return &entities.Claims{
		Username: resp.Claims.Username,
		Subject:  resp.Claims.Subject,
		Rights:   resp.Claims.Rights,
		Audience: resp.Claims.Audience,
		Issuer:   resp.Claims.Issuer,
	}, nil
}
