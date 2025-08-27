package common

import (
	"context"
	"log/slog"

	"github.com/parta4ok/kvs/auth/internal/entities"
	"github.com/pkg/errors"
)

var (
	_ entities.Command = (*AddGroupCommand)(nil)
)

type AddGroupCommand struct {
	storage   Storage
	generator IDGenerator

	title    string
	linkedID string
	ctx      context.Context
}

func NewAddGroupCommand(ctx context.Context, storage Storage, generator IDGenerator,
	title, linkedID string) (*AddGroupCommand, error) {
	if storage == nil {
		return nil, errors.Wrap(entities.ErrInvalidParam, "storage not set")
	}

	if generator == nil {
		return nil, errors.Wrap(entities.ErrInvalidParam, "generator not set")
	}

	if title == "" || linkedID == "" {
		return nil, errors.Wrap(entities.ErrInvalidParam, "linkedID or group title not set")
	}

	return &AddGroupCommand{
		storage:   storage,
		generator: generator,

		ctx:      ctx,
		linkedID: linkedID,
		title:    title,
	}, nil
}

func (command *AddGroupCommand) Exec() (*entities.CommandResult, error) {
	slog.Info("AddGroupCommand exec started")

	gid, err := command.generator.Generate(command.ctx)
	if err != nil {
		err := errors.Wrap(err, "generate failure")
		slog.Error(err.Error())
		return nil, err
	}

	if err := command.storage.AddGroup(command.ctx, gid, command.title, command.linkedID); err != nil {
		err := errors.Wrap(err, "add group failure")
		slog.Error(err.Error())
		return nil, err
	}

	return &entities.CommandResult{
		Success: true,
		Message: gid,
	}, nil
}
