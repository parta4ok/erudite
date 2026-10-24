package authservice

import (
	"context"
	"log/slog"

	authv1 "github.com/parta4ok/kvs/api/grpc/v1"
	"github.com/parta4ok/kvs/question/internal/entities"
	"github.com/parta4ok/kvs/toolkit/pkg/accessor"
	"github.com/parta4ok/kvs/toolkit/pkg/auth/client"
	"github.com/parta4ok/kvs/toolkit/pkg/middleware"
	"github.com/parta4ok/kvs/toolkit/pkg/tracer"
	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	_ middleware.Introspector = (*AuthService)(nil)
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

func (srv *AuthService) Introspect(ctx context.Context, jwt string) (*accessor.Claims, error) {
	slog.Info("Introspect started")
	ctx, span, cancel := tracer.Start(ctx, "AuthServiceIntrospectSpan")
	defer cancel()

	req := &authv1.IntrospectRequest{
		Token: jwt,
	}

	resp, err := srv.client.Introspect(ctx, req)
	if err != nil {
		if status.Code(err) == codes.Unauthenticated {
			err = errors.Wrap(middleware.ErrUnauthorized, "introspect failure")
			slog.Error("introspect failure", "error", err.Error())
			span.SetError(err)
			return nil, err
		}
		err = errors.Wrapf(entities.ErrInternal, "introspect failure: %v", err)
		slog.Error("introspect failure", "error", err.Error())
		span.SetError(err)
		return nil, err
	}

	slog.Info("Introspect completed")

	return &accessor.Claims{
		Username: resp.Claims.Username,
		Subject:  resp.Claims.Subject,
		Rights:   resp.Claims.Rights,
		Audience: resp.Claims.Audience,
		Issuer:   resp.Claims.Issuer,
		GroupID:  resp.Claims.GroupId,
	}, nil
}
