package command

import (
	"context"
	"log/slog"

	"github.com/pkg/errors"

	"github.com/parta4ok/kvs/auth/internal/cases/common"
	"github.com/parta4ok/kvs/auth/internal/entities"
	"github.com/parta4ok/kvs/toolkit/pkg/tracing"
)

var (
	_ entities.Command = (*GetLinkedUsersCommand)(nil)
)

type GetLinkedUsersCommand struct {
	storage common.Storage

	userID string
	ctx    context.Context
}

func NewGetLinkedUsersCommand(ctx context.Context, storage common.Storage, userID string,
) (*GetLinkedUsersCommand, error) {
	if storage == nil {
		return nil, errors.Wrap(entities.ErrInvalidParam, "storage not set")
	}

	if userID == "" {
		return nil, errors.Wrap(entities.ErrInvalidParam, "user id id empty")
	}

	return &GetLinkedUsersCommand{
		storage: storage,

		ctx:    ctx,
		userID: userID,
	}, nil
}

func (command *GetLinkedUsersCommand) Exec() (*entities.CommandResult, error) {
	slog.Info("GetLinkedUserCommand exec started", slog.String("student_id", command.userID))
	ctx, span, cancel := tracing.GlobalTracer().Start(command.ctx, "GetLinkedUsersCommandExecSpan")
	defer cancel()

	linkedUsers, err := command.storage.GetLinkedUsers(ctx, command.userID)
	if err != nil {
		err = errors.Wrap(err, "get linked users failure")
		slog.Error(err.Error())
		span.SetError(err, "get linked users failure")
		return nil, err
	}

	slog.Info("GetLinkedUsersCommand exec completed")
	return &entities.CommandResult{Success: true, Payload: linkedUsers}, nil
}
