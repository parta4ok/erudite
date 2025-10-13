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
	_ entities.Command = (*GetMentorGroupsCommand)(nil)
)

type GetMentorGroupsCommand struct {
	storage  common.Storage
	ctx      context.Context
	mentorID string
}

func NewGetMentorGroupsCommand(ctx context.Context, storage common.Storage, mentorID string) (
	*GetMentorGroupsCommand, error) {
	if storage == nil {
		return nil, errors.Wrap(entities.ErrInvalidParam, "storage not set")
	}

	if mentorID == "" {
		return nil, errors.Wrap(entities.ErrInvalidParam, "mentorID not set")
	}

	return &GetMentorGroupsCommand{
		storage:  storage,
		ctx:      ctx,
		mentorID: mentorID,
	}, nil
}

func (command *GetMentorGroupsCommand) Exec() (*entities.CommandResult, error) {
	slog.Info("AddUserCommand exec started")
	ctx, span, cancel := tracing.GlobalTracer().Start(command.ctx, "GetMentorGroupsCommandExecSpan")
	defer cancel()

	groups, err := command.storage.GetMentorGroups(ctx, command.mentorID)
	if err != nil {
		err = errors.Wrap(err, "get mentor groups")
		slog.Error(err.Error())
		span.SetError(err, "get mentor groups")
		return nil, err
	}

	result := &entities.CommandResult{
		Success: true,
		Payload: groups,
	}
	slog.Info("GetMentorGroupsCommand exec finished")
	return result, nil
}
