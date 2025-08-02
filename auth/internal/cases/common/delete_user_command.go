package common

import (
	"context"
	"log/slog"

	"github.com/parta4ok/kvs/auth/internal/entities"
	"github.com/pkg/errors"
)

var (
	_ entities.Command = (*DeleteUserCommand)(nil)
)

type DeleteUserCommand struct {
	storage Storage

	ctx    context.Context
	userID string
}

func NewDeleteUserCommand(ctx context.Context, storage Storage, userID string) (
	*DeleteUserCommand, error) {
	if storage == nil {
		return nil, errors.Wrap(entities.ErrInvalidParam, "storage not set")
	}

	if userID == "" {
		return nil, errors.Wrap(entities.ErrInvalidParam, "userID is incorrect")
	}

	return &DeleteUserCommand{
		storage: storage,
		ctx:     ctx,
		userID:  userID,
	}, nil
}

func (command *DeleteUserCommand) Exec() (*entities.CommandResult, error) {
	slog.Info("DeleteUserCommand started")

	if err := command.storage.RemoveUser(command.ctx, command.userID); err != nil {
		err = errors.Wrap(err, "RemoveUser failure")
		slog.Error(err.Error())
		return nil, err
	}

	slog.Info("DeleteUserCommand exec completed")
	return &entities.CommandResult{
		Success: true,
	}, nil
}
