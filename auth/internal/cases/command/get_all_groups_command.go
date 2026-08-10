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
	_ entities.Command = (*GetAllGroupsCommand)(nil)
)

type GetAllGroupsCommand struct {
	storage common.Storage
}

func NewGetAllGroupsCommand(storage common.Storage) *GetAllGroupsCommand {
	return &GetAllGroupsCommand{
		storage: storage,
	}
}

func (command *GetAllGroupsCommand) Exec(ctx context.Context) (*entities.CommandResult, error) {
	slog.Info("GetAllGroupsCommand exec started")
	ctx, span, cancel := tracer.Start(ctx, "GetAllGroupsCommandExecSpan")
	defer cancel()

	groups, err := command.storage.GetAllGroups(ctx)
	if err != nil {
		err = errors.Wrap(err, "get all groups")
		slog.Error(err.Error())
		span.SetError(err)
		return nil, err
	}

	result := &entities.CommandResult{
		Success: true,
		Payload: groups,
	}
	slog.Info("GetAllGroupsCommand exec finished")
	return result, nil
}
