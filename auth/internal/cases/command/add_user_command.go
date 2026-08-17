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
	_ entities.Command = (*AddUserCommand)(nil)
)

type AddUserCommand struct {
	storage   common.Storage
	hasher    common.Hasher
	generator common.IDGenerator

	user *entities.User
}

func NewAddUserCommand(storage common.Storage, hasher common.Hasher,
	generator common.IDGenerator, user *entities.User) *AddUserCommand {
	return &AddUserCommand{
		storage:   storage,
		hasher:    hasher,
		generator: generator,

		user: user,
	}
}

//nolint:funlen //ok
func (command *AddUserCommand) Exec(ctx context.Context) (*entities.CommandResult, error) {
	slog.Info("AddUserCommand exec started")
	ctx, span, cancel := tracer.Start(ctx, "AddUserCommandExecSpan")
	defer cancel()

	if command.user == nil {
		return nil, errors.Wrap(entities.ErrInvalidParam, "user not set")
	}

	_, err := command.storage.GetUserByUsername(ctx, command.user.Username)
	if err != nil {
		if !errors.Is(err, entities.ErrNotFound) {
			err = errors.Wrap(err, "get user by user id")
			slog.Error("get user by user id", "error", err.Error())
			span.SetError(err)
			return nil, err
		}
	}

	if err == nil {
		err = errors.Wrapf(entities.ErrAlreadyExists, "user name %s already exists",
			command.user.Username)
		slog.Error("user name %s already exists", "error", err.Error())
		span.SetError(err)
		return nil, err
	}

	userID, err := command.generator.Generate(ctx)
	if err != nil {
		err := errors.Wrap(err, "generate failure")
		slog.Error("generate failure", "error", err.Error())
		span.SetError(err)
		return nil, err
	}

	hash, err := command.hasher.Hash(ctx, command.user.PasswordHash)
	if err != nil {
		err := errors.Wrap(err, "hash password failure")
		slog.Error("hash password failure", "error", err.Error())
		span.SetError(err)
		return nil, err
	}

	user := &entities.User{
		ID:           userID,
		Username:     command.user.Username,
		PasswordHash: hash,
		FullName:     command.user.FullName,
		Rights:       command.user.Rights,
		Contacts:     command.user.Contacts,
		GroupID:      command.user.GroupID,
	}

	if err := command.storage.StoreUser(ctx, user); err != nil {
		err = errors.Wrap(err, "store user failure")
		slog.Error("store user failure", "error", err.Error())
		span.SetError(err)
		return nil, err
	}

	slog.Info("AddUserCommand exec completed")
	return &entities.CommandResult{Success: true, Message: user.ID}, nil
}
