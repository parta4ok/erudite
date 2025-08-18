package common

import (
	"context"
	"log/slog"

	"github.com/pkg/errors"

	"github.com/parta4ok/kvs/auth/internal/entities"
)

var (
	_ entities.Command = (*GetUserCommand)(nil)
)

type GetUserCommand struct {
	storage Storage

	userID string
	ctx    context.Context
}

func NewGetUserCommand(ctx context.Context, storage Storage, userID string,
) (*GetUserCommand, error) {
	if storage == nil {
		return nil, errors.Wrap(entities.ErrInvalidParam, "storage not set")
	}

	if userID == "" {
		return nil, errors.Wrap(entities.ErrInvalidParam, "user id id empty")
	}

	return &GetUserCommand{
		storage: storage,

		ctx:    ctx,
		userID: userID,
	}, nil
}

func (command *GetUserCommand) Exec() (*entities.CommandResult, error) {
	slog.Info("GetUserCommand exec started")

	user, err := command.storage.GetUserByID(command.ctx, command.userID)
	if err != nil {
		err = errors.Wrap(err, "get user failure")
		slog.Error(err.Error())
		return nil, err
	}

	slog.Info("GetUserCommand exec completed")
	return &entities.CommandResult{Success: true, Payload: user}, nil
}
