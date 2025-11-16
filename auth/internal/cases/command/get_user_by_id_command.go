package command

import (
	"context"
	"log/slog"

	"github.com/parta4ok/kvs/auth/internal/cases/common"
	"github.com/parta4ok/kvs/auth/internal/entities"
	"github.com/parta4ok/kvs/toolkit/pkg/tracing"
	"github.com/pkg/errors"
)

var (
	_ entities.Command = (*IntrospectCommand)(nil)
)

type GetUserByIDCommand struct {
	storage common.Storage

	ctx    context.Context
	userID string
}

func NewGetUserByIDCommand(ctx context.Context, storage common.Storage,
	userID string) (*GetUserByIDCommand, error) {

	return &GetUserByIDCommand{
		storage: storage,

		ctx:    ctx,
		userID: userID,
	}, nil
}

func (command *GetUserByIDCommand) Exec() (*entities.CommandResult, error) {
	slog.Info("IntrospectCommand exec started")
	ctx, span, cancel := tracing.GlobalTracer().Start(command.ctx, "GetUserByIDCommandSpan")
	defer cancel()

	user, err := command.storage.GetUserByID(ctx, command.userID)
	if err != nil {
		err = errors.Wrap(err, "GetUserByID")
		slog.Error(err.Error())
		span.SetError(err, "GetUserByID")
		return nil, err
	}

	slog.Info("IntrospectCommand exec completed")
	return &entities.CommandResult{
		Success: true,
		Payload: user,
	}, nil
}
