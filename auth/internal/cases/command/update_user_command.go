package command

import (
	"context"
	"log/slog"

	"github.com/pkg/errors"

	"github.com/parta4ok/kvs/auth/internal/cases/common"
	"github.com/parta4ok/kvs/auth/internal/entities"
	"github.com/parta4ok/kvs/toolkit/pkg/tracer"
)

var (
	_ entities.Command = (*UpdateUserCommand)(nil)
)

type UpdateUserCommand struct {
	storage common.Storage
	hasher  common.Hasher

	userUpdate *entities.UserUpdate
}

func NewUpdateUserCommand(storage common.Storage, hasher common.Hasher, user *entities.UserUpdate,
) *UpdateUserCommand {
	return &UpdateUserCommand{
		storage: storage,
		hasher:  hasher,

		userUpdate: user,
	}
}

func (command *UpdateUserCommand) Exec(ctx context.Context) (*entities.CommandResult, error) {
	slog.Info("UpdateUserCommand exec started")
	ctx, span, cancel := tracer.Start(ctx, "UpdateUserCommandExecSpan")
	defer cancel()

	if command.userUpdate == nil {
		return nil, errors.Wrap(entities.ErrInvalidParam, "user update data is empty")
	}

	if command.userUpdate.PasswordHash != nil {
		hash, err := command.hasher.Hash(ctx, *command.userUpdate.PasswordHash)
		if err != nil {
			err := errors.Wrap(err, "hash password failure")
			slog.Error("hash password failure", "error", err)
			span.SetError(err)
			return nil, err
		}
		*command.userUpdate.PasswordHash = hash
	}

	if err := command.storage.UpdateUser(ctx, command.userUpdate); err != nil {
		err = errors.Wrap(err, "update user failure")
		slog.Error("update user failure", "error", err)
		span.SetError(err)
		return nil, err
	}

	slog.Info("UpdateUserCommand exec completed")
	return &entities.CommandResult{Success: true, Message: command.userUpdate.ID}, nil
}
