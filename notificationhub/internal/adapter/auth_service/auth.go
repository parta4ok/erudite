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

func (srv *AuthService) GetLinkedUsers(ctx context.Context, id string,
) (*entities.LinkedUsers, error) {
	slog.Info("GetLinkedUsers started", slog.String("id", id))

	req := &authv1.LinkedID{
		LinkedID: id,
	}

	linkedUsers, err := srv.client.GetLinkedUsers(ctx, req)
	if err != nil {
		err = errors.Wrapf(entities.ErrInternal, "GetLinkedUsers failure: %v", err)
		slog.Error(err.Error())
		return nil, err
	}

	if linkedUsers.Error.Message != "" {
		err := errors.Wrapf(entities.ErrInternal, "error message: %s", linkedUsers.Error.Message)
		slog.Error(err.Error())
		return nil, err
	}

	if linkedUsers.Recipient == nil || linkedUsers.Student == nil {
		err := errors.Wrap(entities.ErrInternal, "nil users info")
		slog.Error(err.Error())
		return nil, err
	}

	slog.Info("GetLinkedUsers completed")

	return &entities.LinkedUsers{
		Recipient: &entities.User{
			ID:       linkedUsers.Recipient.Id,
			Name:     linkedUsers.Recipient.Username,
			Fullname: linkedUsers.Recipient.Fullname,
			Rights:   linkedUsers.Recipient.Rights,
			Contacts: linkedUsers.Recipient.Contacts,
			GroupID:  linkedUsers.Recipient.GroupId,
		},
		Student: &entities.User{
			ID:       linkedUsers.Student.Id,
			Name:     linkedUsers.Student.Username,
			Fullname: linkedUsers.Student.Fullname,
			Rights:   linkedUsers.Student.Rights,
			Contacts: linkedUsers.Student.Contacts,
			GroupID:  linkedUsers.Student.GroupId,
		},
	}, nil
}
