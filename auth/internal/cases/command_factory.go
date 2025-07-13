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

	return factory, nil
}

func (cf *CommandFactory) NewIntrospectedCommand(ctx context.Context, userID string, jwt string,
) (entities.Command, error) {
	return common.NewIntrospectCommand(ctx, userID, jwt, cf.storage, cf.jwtProvider)
}

func (cf *CommandFactory) NewSignInCommand(ctx context.Context, userName string, password string,
) (entities.Command, error) {
	return common.NewSignInCommand(ctx, userName, password, cf.storage, cf.jwtProvider)
}
