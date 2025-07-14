package common

import (
	"context"
	"log/slog"
	"slices"

	"github.com/parta4ok/kvs/auth/internal/entities"
	"github.com/pkg/errors"
)

type IntrospectCommand struct {
	storage     Storage
	jwtProvider JWTProvider

	ctx    context.Context
	userID string
	jwt    string
}

func NewIntrospectCommand(ctx context.Context, userID string, jwt string, storage Storage,
	provider JWTProvider) (*IntrospectCommand, error) {
	if jwt == "" {
		return nil, errors.Wrap(entities.ErrInvalidJWT, "jwt is required")
	}

	if userID == "" {
		return nil, errors.Wrap(entities.ErrInvalidParam, "user ID is required")
	}

	return &IntrospectCommand{
		storage:     storage,
		jwtProvider: provider,

		ctx:    ctx,
		userID: userID,
		jwt:    jwt,
	}, nil
}

func (command *IntrospectCommand) Exec() (*entities.CommandResult, error) {
	slog.Info("IntrospectCommand exec started")

	user, err := command.storage.GetUserByID(command.ctx, command.userID)
	if err != nil {
		err = errors.Wrap(err, "GetUserByID")
		slog.Error(err.Error())
		return nil, err
	}

	userClaims, err := command.jwtProvider.Introspect(command.jwt)
	if err != nil {
		err = errors.Wrap(err, "Introspect")
		slog.Error(err.Error())
		return nil, err
	}

	for _, reqRight := range userClaims.Rights {
		if !slices.Contains(user.Rights, reqRight) {
			err := errors.Wrapf(entities.ErrForbidden, "not enough rights")
			slog.Error(err.Error())
			return nil, err
		}
	}

	if userClaims.Subject != command.userID {
		err := errors.Wrapf(entities.ErrForbidden, "user ID mismatch")
		slog.Error(err.Error())
		return nil, err
	}

	slog.Info("IntrospectCommand exec completed")
	return &entities.CommandResult{Success: true}, nil
}
