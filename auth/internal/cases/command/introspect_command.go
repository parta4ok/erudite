package command

import (
	"context"
	"log/slog"
	"slices"

	"github.com/parta4ok/kvs/auth/internal/cases/common"
	"github.com/parta4ok/kvs/auth/internal/entities"
	"github.com/parta4ok/kvs/toolkit/pkg/tracing"
	"github.com/pkg/errors"
)

var (
	_ entities.Command = (*IntrospectCommand)(nil)
)

type IntrospectCommand struct {
	storage     common.Storage
	jwtProvider common.JWTProvider

	jwt string
}

func NewIntrospectCommand(jwt string, storage common.Storage, provider common.JWTProvider,
) *IntrospectCommand {
	return &IntrospectCommand{
		storage:     storage,
		jwtProvider: provider,

		jwt: jwt,
	}
}

func (command *IntrospectCommand) Exec(ctx context.Context) (*entities.CommandResult, error) {
	slog.Info("IntrospectCommand exec started")
	ctx, span, cancel := tracing.GlobalTracer().Start(ctx, "IntrospectCommandExecSpan")
	defer cancel()

	if command.jwt == "" {
		return nil, errors.Wrap(entities.ErrInvalidJWT, "jwt is required")
	}
	
	userClaims, err := command.jwtProvider.Introspect(command.jwt)
	if err != nil {
		err = errors.Wrap(err, "Introspect")
		slog.Error(err.Error())
		span.SetError(err, "Introspect")
		return nil, err
	}

	user, err := command.storage.GetUserByID(ctx, userClaims.Subject)
	if err != nil {
		err = errors.Wrap(err, "GetUserByID")
		slog.Error(err.Error())
		span.SetError(err, "GetUserByID")
		return nil, err
	}

	for _, reqRight := range userClaims.Rights {
		if !slices.Contains(user.Rights, reqRight) {
			err := errors.Wrapf(entities.ErrForbidden, "not enough rights")
			slog.Error(err.Error())
			span.SetError(err, "not enough rights")
			return nil, err
		}
	}

	slog.Info("IntrospectCommand exec completed")
	return &entities.CommandResult{
		Success: true,
		Payload: userClaims,
	}, nil
}
