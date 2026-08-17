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
	_ entities.Command = (*GetAllUsersCommand)(nil)
)

type GetAllUsersCommand struct {
	storage common.Storage
}

func NewGetAllUsersCommand(storage common.Storage) *GetAllUsersCommand {
	return &GetAllUsersCommand{
		storage: storage,
	}
}

func (command *GetAllUsersCommand) Exec(ctx context.Context) (*entities.CommandResult, error) {
	slog.Info("GetAllUsersCommand exec started")
	ctx, span, cancel := tracer.Start(ctx, "GetAllUsersCommandExecSpan")
	defer cancel()

	users, err := command.storage.GetAllUsers(ctx)
	if err != nil {
		err = errors.Wrap(err, "get all users")
		slog.Error("get all users", "error", err.Error())
		span.SetError(err)
		return nil, err
	}

	result := &entities.CommandResult{
		Success: true,
		Payload: users,
	}
	slog.Info("GetAllUsersCommand exec finished")
	return result, nil
}
