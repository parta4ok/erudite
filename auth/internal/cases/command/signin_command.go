package command

import (
	"context"
	"log/slog"

	"github.com/parta4ok/kvs/auth/internal/cases/common"
	"github.com/parta4ok/kvs/auth/internal/entities"
	"github.com/parta4ok/kvs/toolkit/pkg/tracing"
	"github.com/pkg/errors"
)

var (
	_ entities.Command = (*SignInCommand)(nil)
)

type SignInCommand struct {
	storage     common.Storage
	jwtProvider common.JWTProvider
	hasher      common.Hasher

	userName string
	password string
}

func NewSignInCommand(storage common.Storage, provider common.JWTProvider,
	hasher common.Hasher, userName string, password string) *SignInCommand {
	return &SignInCommand{
		storage:     storage,
		jwtProvider: provider,
		hasher:      hasher,

		userName: userName,
		password: password,
	}
}

func (command *SignInCommand) Exec(ctx context.Context) (*entities.CommandResult, error) {
	slog.Info("SignIn command started")
	ctx, span, cancel := tracing.GlobalTracer().Start(ctx, "SignInCommandExecSpan")
	defer cancel()

	if command.userName == "" {
		return nil, errors.Wrap(entities.ErrInvalidParam, "username is required")
	}

	if command.password == "" {
		return nil, errors.Wrap(entities.ErrInvalidParam, "password is required")
	}

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
