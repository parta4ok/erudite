package cases

import (
	"context"

	"github.com/parta4ok/kvs/auth/internal/cases/common"
	"github.com/parta4ok/kvs/auth/internal/entities"
	"github.com/pkg/errors"
)

type CommandFactory struct {
	storage     common.Storage
	jwtProvider common.JWTProvider
	hasher      common.Hasher
	idGenerator common.IDGenerator
}

type CommandFactoryOption func(*CommandFactory)

func WithStorage(storage common.Storage) CommandFactoryOption {
	return func(cf *CommandFactory) {
		cf.storage = storage
	}
}

func WithJWTProvider(jwtProvider common.JWTProvider) CommandFactoryOption {
	return func(cf *CommandFactory) {
		cf.jwtProvider = jwtProvider
	}
}

func WithHasher(hasher common.Hasher) CommandFactoryOption {
	return func(cf *CommandFactory) {
		cf.hasher = hasher
	}
}

func WithIDGenerator(generator common.IDGenerator) CommandFactoryOption {
	return func(cf *CommandFactory) {
		cf.idGenerator = generator
	}
}

func (cf *CommandFactory) setOptions(opts ...CommandFactoryOption) {
	for _, opt := range opts {
		opt(cf)
	}
}

func NewCommandFactory(opts ...CommandFactoryOption) (*CommandFactory, error) {
	factory := &CommandFactory{}

	factory.setOptions(opts...)

	if factory.storage == nil {
		return nil, errors.Wrap(entities.ErrInvalidParam, "storage not set")
	}

	if factory.jwtProvider == nil {
		return nil, errors.Wrap(entities.ErrInvalidParam, "jwt provider not set")
	}

	if factory.hasher == nil {
		return nil, errors.Wrap(entities.ErrInvalidParam, "hasher not set")
	}

	if factory.idGenerator == nil {
		return nil, errors.Wrap(entities.ErrInvalidParam, "id generator not set")
	}

	return factory, nil
}

func (cf *CommandFactory) NewIntrospectedCommand(
	ctx context.Context,
	jwt string,
) (entities.Command, error) {
	return common.NewIntrospectCommand(ctx, jwt, cf.storage, cf.jwtProvider)
}

func (cf *CommandFactory) NewSignInCommand(
	ctx context.Context,
	userName string,
	password string,
) (entities.Command, error) {
	return common.NewSignInCommand(ctx, cf.storage, cf.jwtProvider, cf.hasher, userName, password)
}

func (cf *CommandFactory) NewAddUserCommand(
	ctx context.Context,
	login string,
	password string,
	rights []string,
	contacts map[string]string,
) (entities.Command, error) {
	return common.NewAddUserCommand(ctx, cf.storage, cf.hasher, cf.idGenerator, login, password,
		rights, contacts)
}

func (cf *CommandFactory) NewDeleteUserCommand(
	ctx context.Context,
	userID string,
) (entities.Command, error) {
	return common.NewDeleteUserCommand(ctx, cf.storage, userID)
}
