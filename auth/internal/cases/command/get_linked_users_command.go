package command

import (
	"context"
	"log/slog"

	"github.com/pkg/errors"

	"github.com/parta4ok/kvs/auth/internal/cases/common"
	"github.com/parta4ok/kvs/auth/internal/entities"
	"github.com/parta4ok/kvs/toolkit/pkg/tracer"
)

var (
	_ entities.Command = (*GetLinkedUsersCommand)(nil)
)

type GetLinkedUsersCommand struct {
	storage common.Storage

	userID string
}

func NewGetLinkedUsersCommand(storage common.Storage, userID string) *GetLinkedUsersCommand {
	return &GetLinkedUsersCommand{
		storage: storage,

		userID: userID,
	}
}

func (command *GetLinkedUsersCommand) Exec(ctx context.Context) (*entities.CommandResult, error) {
	slog.Info("GetLinkedUserCommand exec started", slog.String("student_id", command.userID))
	ctx, span, cancel := tracer.Start(ctx, "GetLinkedUsersCommandExecSpan")
	defer cancel()

	if command.userID == "" {
		return nil, errors.Wrap(entities.ErrInvalidParam, "user id id empty")
	}

	linkedUsers, err := command.storage.GetLinkedUsers(ctx, command.userID)
	if err != nil {
		err = errors.Wrap(err, "get linked users failure")
		slog.Error(err.Error())
		span.SetError(err)
		return nil, err
	}

	slog.Info("GetLinkedUsersCommand exec completed")
	return &entities.CommandResult{Success: true, Payload: linkedUsers}, nil
}
