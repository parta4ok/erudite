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
	ctx, span, cancel := tracer.Start(ctx, "IntrospectCommandExecSpan")
	defer cancel()

	if command.jwt == "" {
		return nil, errors.Wrap(entities.ErrInvalidJWT, "jwt is required")
	}

	userClaims, err := command.jwtProvider.Introspect(command.jwt)
	if err != nil {
		err = errors.Wrap(err, "Introspect")
		slog.Error("Introspect", "error", err.Error())
		span.SetError(err)
		return nil, err
	}

	user, err := command.storage.GetUserByID(ctx, userClaims.Subject)
	if err != nil {
		err = errors.Wrap(err, "GetUserByID")
		slog.Error("GetUserByID", "error", err.Error())
		span.SetError(err)
		return nil, err
	}

	slog.Info("IntrospectCommand exec completed")
	return &entities.CommandResult{
		Success: true,
		Payload: &entities.UserClaims{
			Subject:  user.ID,
			Username: user.Username,
			Rights:   user.Rights,
			GroupID:  user.GroupID,
			Issuer:   userClaims.Issuer,
			Audience: userClaims.Audience,
		},
	}, nil
}
