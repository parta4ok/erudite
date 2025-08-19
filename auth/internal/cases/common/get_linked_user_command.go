package common

import (
	"context"
	"log/slog"

	"github.com/pkg/errors"

	"github.com/parta4ok/kvs/auth/internal/entities"
)

var (
	_ entities.Command = (*GetLinkedUserCommand)(nil)
)

type GetLinkedUserCommand struct {
	storage Storage

	userID string
	ctx    context.Context
}

func NewGetLinkedUserCommand(ctx context.Context, storage Storage, userID string,
) (*GetLinkedUserCommand, error) {
	if storage == nil {
		return nil, errors.Wrap(entities.ErrInvalidParam, "storage not set")
	}

	if userID == "" {
		return nil, errors.Wrap(entities.ErrInvalidParam, "user id id empty")
	}

	return &GetLinkedUserCommand{
		storage: storage,

		ctx:    ctx,
		userID: userID,
	}, nil
}

func (command *GetLinkedUserCommand) Exec() (*entities.CommandResult, error) {
	slog.Info("GetLinkedUserCommand exec started")

	user, err := command.storage.GetUserByLinkedID(command.ctx, command.userID)
	if err != nil {
		err = errors.Wrap(err, "get user failure")
		slog.Error(err.Error())
		return nil, err
	}

	slog.Info("GetLinkedUserCommand exec completed")
	return &entities.CommandResult{Success: true, Payload: user}, nil
}
