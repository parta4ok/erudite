package common

import (
	"context"
	"log/slog"

	"github.com/parta4ok/kvs/auth/internal/entities"
	"github.com/parta4ok/kvs/toolkit/pkg/tracing"
	"github.com/pkg/errors"
)

var (
	_ entities.Command = (*SignInCommand)(nil)
)

type SignInCommand struct {
	storage     Storage
	jwtProvider JWTProvider
	hasher      Hasher

	ctx      context.Context
	userName string
	password string
}

func NewSignInCommand(ctx context.Context, storage Storage, provider JWTProvider, hasher Hasher,
	userName string, password string) (*SignInCommand, error) {
	if storage == nil {
		return nil, errors.Wrap(entities.ErrInvalidParam, "storage not set")
	}

	if provider == nil {
		return nil, errors.Wrap(entities.ErrInvalidParam, "jwt provider not set")
	}

	if hasher == nil {
		return nil, errors.Wrap(entities.ErrInvalidParam, "hasher not set")
	}

	if userName == "" {
		return nil, errors.Wrap(entities.ErrInvalidParam, "username is required")
	}

	if password == "" {
		return nil, errors.Wrap(entities.ErrInvalidParam, "password is required")
	}

	return &SignInCommand{
		storage:     storage,
		jwtProvider: provider,
		hasher:      hasher,

		ctx:      ctx,
		userName: userName,
		password: password,
	}, nil
}

func (command *SignInCommand) Exec() (*entities.CommandResult, error) {
	slog.Info("SignIn command started")
	ctx, span, cancel := tracing.GlobalTracer().Start(command.ctx, "SignInCommandExecSpan")
	defer cancel()

	user, err := command.storage.GetUserByUsername(ctx, command.userName)
	if err != nil {
		err = errors.Wrap(err, "GetUserByUsername")
		slog.Error(err.Error())
		span.SetError(err, "GetUserByUsername")
		return nil, err
	}

	isHash, err := command.hasher.IsHash(ctx, command.password, user.PasswordHash)
	if err != nil {
		err = errors.Wrap(err, "IsHash failire")
		slog.Error(err.Error())
		span.SetError(err, "IsHash failire")
		return nil, err
	}

	if !isHash {
		err = errors.Wrapf(entities.ErrInvalidPassword, "approvePassword failire: %v", err)
		slog.Error(err.Error())
		span.SetError(err, "approvePassword failire")
		return nil, err
	}

	jwt, err := command.jwtProvider.Generate(user)
	if err != nil {
		err = errors.Wrap(err, "Generate JWT failure")
		slog.Error(err.Error())
		span.SetError(err, "Generate JWT failure")
		return nil, err
	}

	return &entities.CommandResult{Success: true, Message: jwt}, nil
}
