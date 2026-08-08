package command

import (
	"context"
	"log/slog"

	"github.com/parta4ok/kvs/auth/internal/cases/common"
	"github.com/parta4ok/kvs/auth/internal/entities"
	"github.com/parta4ok/kvs/toolkit/pkg/tracer"
	"github.com/pkg/errors"
)

var (
	_ entities.Command = (*DeleteUserCommand)(nil)
)

type DeleteUserCommand struct {
	storage common.Storage

	userID string
}

func NewDeleteUserCommand(storage common.Storage, userID string) *DeleteUserCommand {
	return &DeleteUserCommand{
		storage: storage,

		userID: userID,
	}
}

func (command *DeleteUserCommand) Exec(ctx context.Context) (*entities.CommandResult, error) {
	slog.Info("DeleteUserCommand started")
	ctx, span, cancel := tracer.Start(ctx, "DeleteUserCommandExecSpan")
	defer cancel()

	if command.userID == "" {
		return nil, errors.Wrap(entities.ErrInvalidParam, "userID is incorrect")
	}

	if err := command.storage.RemoveUser(ctx, command.userID); err != nil {
		err = errors.Wrap(err, "RemoveUser failure")
		slog.Error(err.Error())
		span.SetError(err)
		return nil, err
	}

	slog.Info("DeleteUserCommand exec completed")
	return &entities.CommandResult{
		Success: true,
	}, nil
}
