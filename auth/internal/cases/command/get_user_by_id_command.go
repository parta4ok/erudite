//nolint:dupl //ok
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
	_ entities.Command = (*GetUserByIDCommand)(nil)
)

type GetUserByIDCommand struct {
	storage common.Storage

	userID string
}

func NewGetUserByIDCommand(storage common.Storage, userID string) *GetUserByIDCommand {
	return &GetUserByIDCommand{
		storage: storage,

		userID: userID,
	}
}

func (command *GetUserByIDCommand) Exec(ctx context.Context) (*entities.CommandResult, error) {
	slog.Info("GetUserByIDCommand exec started")
	ctx, span, cancel := tracer.Start(ctx, "GetUserByIDCommandSpan")
	defer cancel()

	if command.userID == "" {
		return nil, errors.Wrap(entities.ErrInvalidParam, "user id not set")
	}

	user, err := command.storage.GetUserByID(ctx, command.userID)
	if err != nil {
		err = errors.Wrap(err, "GetUserByID")
		slog.Error(err.Error())
		span.SetError(err)
		return nil, err
	}

	slog.Info("GetUserByIDCommand exec completed")
	return &entities.CommandResult{
		Success: true,
		Payload: user,
	}, nil
}
