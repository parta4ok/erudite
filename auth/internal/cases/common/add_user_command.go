package common

import (
	"context"
	"log/slog"

	"github.com/pkg/errors"

	"github.com/parta4ok/kvs/auth/internal/entities"
)

type AddUserCommand struct {
	storage   Storage
	hasher    Hasher
	generator IDGenerator

	login    string
	password string
	rights   []string
	contacts map[string]string
	ctx      context.Context
}

func NewAddUserCommand(ctx context.Context, storage Storage, hasher Hasher, generator IDGenerator,
	login, password string,
	rights []string, contacts map[string]string) (*AddUserCommand, error) {
	if storage == nil {
		return nil, errors.Wrap(entities.ErrInvalidParam, "storage not set")
	}

	if hasher == nil {
		return nil, errors.Wrap(entities.ErrInvalidParam, "hasher not set")
	}

	if generator == nil {
		return nil, errors.Wrap(entities.ErrInvalidParam, "generator not set")
	}

	if login == "" || password == "" {
		return nil, errors.Wrap(entities.ErrInvalidParam, "login or password is incorrect")
	}

	return &AddUserCommand{
		storage:   storage,
		hasher:    hasher,
		generator: generator,

		ctx:      ctx,
		login:    login,
		password: password,
		rights:   rights,
		contacts: contacts,
	}, nil
}

func (command *AddUserCommand) Exec() (*entities.CommandResult, error) {
	slog.Info("AddUserCommand exec started")

	_, err := command.storage.GetUserByUsername(command.ctx, command.login)
	if err != nil {
		if !errors.Is(err, entities.ErrNotFound) {
			err = errors.Wrap(err, "get user by user id")
			slog.Error(err.Error())
			return nil, err
		}
	}

	if err == nil {
		err = errors.Wrapf(entities.ErrAlreadyExists, "user name %s already exists", command.login)
		slog.Error(err.Error())
		return nil, err
	}

	userID, err := command.generator.Generate(command.ctx)
	if err != nil {
		err := errors.Wrap(err, "generate failure")
		slog.Error(err.Error())
		return nil, err
	}

	hash, err := command.hasher.Hash(command.ctx, command.password)
	if err != nil {
		err := errors.Wrap(err, "hash password failure")
		slog.Error(err.Error())
		return nil, err
	}

	user := &entities.User{
		ID:           userID,
		Username:     command.login,
		PasswordHash: hash,
		Rights:       command.rights,
		Contacts:     command.contacts,
	}

	if err := command.storage.StoreUser(command.ctx, user); err != nil {
		err = errors.Wrap(err, "store user failure")
		slog.Error(err.Error())
		return nil, err
	}

	slog.Info("AddUserCommand exec completed")
	return &entities.CommandResult{Success: true, Message: user.ID}, nil
}
