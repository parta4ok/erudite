package authservice

import (
	"context"
	"log/slog"

	authv1 "github.com/parta4ok/kvs/api/grpc/v1"
	"github.com/parta4ok/kvs/notificationhub/internal/cases"
	"github.com/parta4ok/kvs/notificationhub/internal/entities"

	"github.com/parta4ok/kvs/toolkit/pkg/auth/client"
	"github.com/pkg/errors"
)

var (
	_ cases.AuthClient = (*AuthService)(nil)
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

func (srv *AuthService) GetRecipientByID(ctx context.Context, id string,
) (*entities.Recipient, error) {
	slog.Info("GetRecipientByID started")

	req := &authv1.LinkedID{
		LinkedID: id,
	}

	info, err := srv.client.GetLinkedUser(ctx, req)
	if err != nil {
		err = errors.Wrapf(entities.ErrInternal, "GetLinkedUser failure: %v", err)
		slog.Error(err.Error())
		return nil, err
	}

	if info.Error.Message != "" {
		err := errors.Wrapf(entities.ErrInternal, "error message: %s", info.Error.Message)
		slog.Error(err.Error())
		return nil, err
	}

	if info.UserInfo == nil {
		err := errors.Wrap(entities.ErrInternal, "nil user info")
		slog.Error(err.Error())
		return nil, err
	}

	slog.Info("GetRecipientByID completed")

	return &entities.Recipient{
		ID:       info.UserInfo.Id,
		Contacts: info.UserInfo.Contacts,
	}, nil
}
