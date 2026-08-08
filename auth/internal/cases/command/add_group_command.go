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
	_ entities.Command = (*AddGroupCommand)(nil)
)

type AddGroupCommand struct {
	storage   common.Storage
	generator common.IDGenerator

	title    string
	linkedID string
}

func NewAddGroupCommand(storage common.Storage, generator common.IDGenerator,
	title, linkedID string) *AddGroupCommand {
	return &AddGroupCommand{
		storage:   storage,
		generator: generator,

		linkedID: linkedID,
		title:    title,
	}
}

func (command *AddGroupCommand) Exec(ctx context.Context) (*entities.CommandResult, error) {
	slog.Info("AddGroupCommand exec started")
	ctx, span, cancel := tracer.Start(ctx, "AddGroupCommandExecSpan")
	defer cancel()

	if command.title == "" || command.linkedID == "" {
		return nil, errors.Wrap(entities.ErrInvalidParam, "linkedID or group title not set")
	}

	gid, err := command.generator.Generate(ctx)
	if err != nil {
		err := errors.Wrap(err, "generate failure")
		slog.Error(err.Error())
		span.SetError(err)
		return nil, err
	}

	if err := command.storage.AddGroup(ctx, gid, command.title, command.linkedID); err != nil {
		err := errors.Wrap(err, "add group failure")
		slog.Error(err.Error())
		span.SetError(err)
		return nil, err
	}

	return &entities.CommandResult{
		Success: true,
		Message: gid,
	}, nil
}
