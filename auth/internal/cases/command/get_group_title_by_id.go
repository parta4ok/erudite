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
	_ entities.Command = (*GetGroupTitleByIDCommand)(nil)
)

type GetGroupTitleByIDCommand struct {
	storage common.Storage

	groupID string
}

func NewGetGroupTitleByIDCommand(storage common.Storage, groupID string) *GetGroupTitleByIDCommand {
	return &GetGroupTitleByIDCommand{
		storage: storage,

		groupID: groupID,
	}
}

func (command *GetGroupTitleByIDCommand) Exec(ctx context.Context) (
	*entities.CommandResult, error) {
	slog.Info("GetGroupTitleByIDCommand exec started")
	ctx, span, cancel := tracer.Start(ctx, "GetGroupTitleByIDCommandSpan")
	defer cancel()

	if command.groupID == "" {
		return nil, errors.Wrap(entities.ErrInvalidParam, "group id not set")
	}

	groupID, err := command.storage.GetGroupTitleByID(ctx, command.groupID)
	if err != nil {
		err = errors.Wrap(err, "GetGroupTitleByID")
		slog.Error("GetGroupTitleByID", "error", err)
		span.SetError(err)
		return nil, err
	}

	slog.Info("GetGroupTitleByIDCommand exec completed")
	return &entities.CommandResult{
		Success: true,
		Payload: groupID,
	}, nil
}
