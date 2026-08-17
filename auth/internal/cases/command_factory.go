package cases

import (
	"github.com/parta4ok/kvs/auth/internal/cases/command"
	"github.com/parta4ok/kvs/auth/internal/cases/common"
	"github.com/parta4ok/kvs/auth/internal/entities"
	"github.com/pkg/errors"
)

type CommandFactory struct {
	storage       common.Storage
	jwtProvider   common.JWTProvider
	hasher        common.Hasher
	idGenerator   common.IDGenerator
	messageBroker common.MessageBroker
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

func WithMessageBroker(broker common.MessageBroker) CommandFactoryOption {
	return func(cf *CommandFactory) {
		cf.messageBroker = broker
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

	if factory.messageBroker == nil {
		return nil, errors.Wrap(entities.ErrInvalidParam, "message broker not set")
	}

	return factory, nil
}

func (cf *CommandFactory) NewIntrospectedCommand(jwt string) entities.Command {
	return command.NewIntrospectCommand(jwt, cf.storage, cf.jwtProvider)
}

func (cf *CommandFactory) NewSignInCommand(userName string, password string) entities.Command {
	return command.NewSignInCommand(cf.storage, cf.jwtProvider, cf.hasher, userName, password)
}

func (cf *CommandFactory) NewAddUserCommand(user *entities.User) entities.Command {
	return command.NewAddUserCommand(cf.storage, cf.hasher, cf.idGenerator, user)
}

func (cf *CommandFactory) NewDeleteUserCommand(userID string) entities.Command {
	return command.NewDeleteUserCommand(cf.storage, userID)
}

func (cf *CommandFactory) NewUpdateUserCommand(userUpdate *entities.UserUpdate) entities.Command {
	return command.NewUpdateUserCommand(cf.storage, cf.hasher, userUpdate)
}

func (cf *CommandFactory) NewGetLinkedUsersCommand(userID string) entities.Command {
	return command.NewGetLinkedUsersCommand(cf.storage, userID)
}

func (cf *CommandFactory) NewAddGroupCommand(title string, linkedID string) entities.Command {
	return command.NewAddGroupCommand(cf.storage, cf.idGenerator, title, linkedID)
}

func (cf *CommandFactory) NewGetMentorGroupsCommand(mentorID string) entities.Command {
	return command.NewGetMentorGroupsCommand(cf.storage, mentorID)
}

func (cf *CommandFactory) NewGetUserByIDCommand(userID string) entities.Command {
	return command.NewGetUserByIDCommand(cf.storage, userID)
}

func (cf *CommandFactory) NewGetGroupTitleByIDCommand(groupID string) entities.Command {
	return command.NewGetGroupTitleByIDCommand(cf.storage, groupID)
}

func (cf *CommandFactory) NewDynamicRegisterCommand(userID, provider string) entities.Command {
	return command.NewDynamicRegisterCommand(cf.storage, cf.idGenerator, cf.messageBroker,
		userID, provider)
}

func (cf *CommandFactory) NewGetAllUsersCommand() entities.Command {
	return command.NewGetAllUsersCommand(cf.storage)
}

func (cf *CommandFactory) NewGetAllGroupsCommand() entities.Command {
	return command.NewGetAllGroupsCommand(cf.storage)
}
