package common

import (
	"context"
	"log/slog"

	"github.com/pkg/errors"

	"github.com/parta4ok/kvs/auth/internal/entities"
)

var (
	_ entities.Command = (*UpdateUserCommand)(nil)
)

type UpdateUserCommand struct {
	storage Storage
	hasher  Hasher

	user *entities.User
	ctx  context.Context
}

func NewUpdateUserCommand(ctx context.Context, storage Storage, hasher Hasher, user *entities.User,
) (*UpdateUserCommand, error) {
	if storage == nil {
		return nil, errors.Wrap(entities.ErrInvalidParam, "storage not set")
	}

	if hasher == nil {
		return nil, errors.Wrap(entities.ErrInvalidParam, "hasher not set")
	}

	if user == nil {
		return nil, errors.Wrap(entities.ErrInvalidParam, "user data is empty")
	}

	return &UpdateUserCommand{
		storage: storage,
		hasher:  hasher,

		ctx:  ctx,
		user: user,
	}, nil
}

func (command *UpdateUserCommand) Exec() (*entities.CommandResult, error) {
	slog.Info("UpdateUserCommand exec started")

	if command.user.PasswordHash != "" {
		hash, err := command.hasher.Hash(command.ctx, command.user.PasswordHash)
		if err != nil {
			err := errors.Wrap(err, "hash password failure")
			slog.Error(err.Error())
			return nil, err
		}
		command.user.PasswordHash = hash
	}

	if err := command.storage.UpdateUser(command.ctx, command.user); err != nil {
		err = errors.Wrap(err, "update user failure")
		slog.Error(err.Error())
		return nil, err
	}

	slog.Info("UpdateUserCommand exec completed")
	return &entities.CommandResult{Success: true, Message: command.user.ID}, nil
}
