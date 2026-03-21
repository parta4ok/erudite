package command

import (
	"context"
	"crypto/rand"
	"fmt"
	"log/slog"
	"math/big"
	"slices"
	"time"

	"github.com/parta4ok/kvs/auth/internal/cases/common"
	"github.com/parta4ok/kvs/auth/internal/entities"
	"github.com/parta4ok/kvs/toolkit/pkg/tracing"
	"github.com/pkg/errors"
)

var (
	_ entities.Command = (*DynamicRegisterCommand)(nil)
)

const (
	DefaultDuration = time.Minute * 2
)

var (
	DefaultProviders = []string{"telegram", "email", "max"}
)

type DynamicRegisterCommand struct {
	storage       common.Storage
	generator     common.IDGenerator
	messageBroker common.MessageBroker

	provider string
	userID   string
}

func NewDynamicRegisterCommand(
	storage common.Storage,
	generator common.IDGenerator,
	messageBroker common.MessageBroker,
	userID string,
	provider string,
) *DynamicRegisterCommand {
	return &DynamicRegisterCommand{
		storage:       storage,
		messageBroker: messageBroker,
		generator:     generator,

		userID:   userID,
		provider: provider,
	}
}

//nolint:funlen //ok
func (cmd *DynamicRegisterCommand) Exec(ctx context.Context) (*entities.CommandResult, error) {
	slog.Info("DynamicRegisterCommand started")
	ctx, span, cancel := tracing.GlobalTracer().Start(ctx, "DynamicRegisterCommandExecSpan")
	defer cancel()

	if cmd.userID == "" {
		err := errors.Wrap(entities.ErrInvalidParam, "userID is incorrect")
		span.SetError(err, "userID is incorrect")
		return nil, err
	}

	existingUser, err := cmd.storage.GetUserByUsername(ctx, cmd.userID)
	if err != nil {
		if !errors.Is(err, entities.ErrNotFound) {
			err := errors.Wrapf(err, "get user by name '%s' failed", cmd.userID)
			span.SetError(err, "get user by name failed")
			return nil, err
		}
	}

	if existingUser != nil {
		err := errors.Wrapf(entities.ErrAlreadyExists, "user with name '%s' already exist", cmd.userID)
		span.SetError(err, "user with ID already exist")
		return nil, err
	}

	if !slices.Contains(DefaultProviders, cmd.provider) {
		err := errors.Wrapf(entities.ErrInvalidParam, "provider %s is not is not avaliable", cmd.provider)
		span.SetError(err, "provider is not is not avaliable")
		return nil, err
	}

	combination, err := rand.Int(rand.Reader, big.NewInt(10000))
	if err != nil {
		err := errors.Wrapf(entities.ErrInternal, "generateted big int failure: %v", err)
		span.SetError(err, "generateted big int failure")
		return nil, err
	}

	code := fmt.Sprintf("%04d", combination.Int64())
	user, err := entities.NewDynamicUser(cmd.userID, cmd.provider)
	if err != nil {
		err := errors.Wrap(err, "creating dynamic user failure")
		span.SetError(err, "creating dynamic user failure")
		return nil, err
	}

	event, err := entities.NewBaseEvent(entities.NotificationAboutShortPassword, []byte(code), user)
	if err != nil {
		err := errors.Wrap(err, "creating base event failure")
		span.SetError(err, "creating base event failure")
		return nil, err
	}

	if err := cmd.messageBroker.SendEvent(ctx, event); err != nil {
		err := errors.Wrap(err, "send event failure")
		span.SetError(err, "send event failure")
		return nil, err
	}

	registrationID, err := cmd.generator.Generate(ctx)
	if err != nil {
		err := errors.Wrap(err, "generateted sessionID failure")
		span.SetError(err, "generateted sessionID failure")
		return nil, err
	}

	registration, err := entities.NewDynamicRegistrationParameters(registrationID, code,
		cmd.userID, cmd.provider, DefaultDuration)
	if err != nil {
		err := errors.Wrap(err, "new dynamic registration failure")
		span.SetError(err, "new dynamic registration failure")
		return nil, err
	}

	if err := cmd.storage.StoreDynamicRegistrations(ctx, registration); err != nil {
		err := errors.Wrap(err, "store dynamic registration failure")
		span.SetError(err, "store dynamic registration failure")
		return nil, err
	}

	return &entities.CommandResult{
		Success: true,
	}, nil
}
