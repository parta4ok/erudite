package command

import (
	"context"
	"log/slog"

	"github.com/pkg/errors"

	"github.com/parta4ok/kvs/auth/internal/cases/common"
	"github.com/parta4ok/kvs/auth/internal/entities"
	"github.com/parta4ok/kvs/toolkit/pkg/tracing"
)

var (
	_ entities.Command = (*UpdateUserCommand)(nil)
)

type UpdateUserCommand struct {
	storage common.Storage
	hasher  common.Hasher

	user *entities.User
}

func NewUpdateUserCommand(storage common.Storage, hasher common.Hasher, user *entities.User,
) *UpdateUserCommand {
	return &UpdateUserCommand{
		storage: storage,
		hasher:  hasher,

		user: user,
	}
}

func (command *UpdateUserCommand) Exec(ctx context.Context) (*entities.CommandResult, error) {
	slog.Info("UpdateUserCommand exec started")
	ctx, span, cancel := tracing.GlobalTracer().Start(ctx, "UpdateUserCommandExecSpan")
	defer cancel()

	if command.user == nil {
		return nil, errors.Wrap(entities.ErrInvalidParam, "user data is empty")
	}

	if command.user.PasswordHash != "" {
		hash, err := command.hasher.Hash(ctx, command.user.PasswordHash)
		if err != nil {
			err := errors.Wrap(err, "hash password failure")
			slog.Error(err.Error())
			span.SetError(err, "hash password failure")
			return nil, err
		}
		command.user.PasswordHash = hash
	}

	if err := command.storage.UpdateUser(ctx, command.user); err != nil {
		err = errors.Wrap(err, "update user failure")
		slog.Error(err.Error())
		span.SetError(err, "update user failure")
		return nil, err
	}

	slog.Info("UpdateUserCommand exec completed")
	return &entities.CommandResult{Success: true, Message: command.user.ID}, nil
}
