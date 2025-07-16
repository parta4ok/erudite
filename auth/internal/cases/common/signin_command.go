package common

import (
	"context"
	"log/slog"

	"github.com/parta4ok/kvs/auth/internal/entities"
	"github.com/pkg/errors"
	"golang.org/x/crypto/bcrypt"
)

type SignInCommand struct {
	storage     Storage
	jwtProvider JWTProvider

	ctx      context.Context
	userName string
	password string
}

func NewSignInCommand(ctx context.Context, userName string, password string, storage Storage,
	provider JWTProvider) (*SignInCommand, error) {
	if userName == "" {
		return nil, errors.Wrap(entities.ErrInvalidParam, "username is required")
	}
	if password == "" {
		return nil, errors.Wrap(entities.ErrInvalidParam, "password is required")
	}
	if storage == nil {
		return nil, errors.Wrap(entities.ErrInvalidParam, "storage not set")
	}
	if provider == nil {
		return nil, errors.Wrap(entities.ErrInvalidParam, "jwt provider not set")
	}

	return &SignInCommand{
		storage:     storage,
		jwtProvider: provider,

		ctx:      ctx,
		userName: userName,
		password: password,
	}, nil
}

func (command *SignInCommand) Exec() (*entities.CommandResult, error) {
	slog.Info("SignIn command started")

	user, err := command.storage.GetUserByUsername(command.ctx, command.userName)
	if err != nil {
		err = errors.Wrap(err, "GetUserByUsername")
		slog.Error(err.Error())
		return nil, err
	}

	if err := command.approvePassword(command.password, user.PasswordHash); err != nil {
		err = errors.Wrapf(entities.ErrInvalidPassword, "approvePassword failire: %v", err)
		slog.Error(err.Error())
		return nil, err
	}

	jwt, err := command.jwtProvider.Generate(user)
	if err != nil {
		err = errors.Wrap(err, "Generate JWT failure")
		slog.Error(err.Error())
		return nil, err
	}

	return &entities.CommandResult{Success: true, Message: jwt}, nil
}

func (c *SignInCommand) approvePassword(reqPass string, hashPass string) error {
	return bcrypt.CompareHashAndPassword([]byte(hashPass), []byte(reqPass))
}
