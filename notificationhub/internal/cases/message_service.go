package cases

import (
	"context"
	"log/slog"

	"github.com/parta4ok/kvs/notificationhub/internal/entities"
	"github.com/parta4ok/kvs/toolkit/pkg/tracer"
	"github.com/pkg/errors"
)

type MessageService struct {
	notifier Notifier
}

func NewMessageService(notifier Notifier) (*MessageService, error) {
	if notifier == nil {
		return nil, errors.Wrap(entities.ErrInvalidParam, "notifier is nil")
	}

	return &MessageService{
		notifier: notifier,
	}, nil
}

func (ms *MessageService) SendMessage(ctx context.Context, message entities.Event) error {
	ctx, span, cancel := tracer.Start(ctx, "SendMessageSpan")
	defer cancel()

	if message == nil {
		err := errors.Wrap(entities.ErrInvalidParam, "message is nil")
		slog.Error("message is nil", "error", err.Error())
		span.SetError(err)
		return err
	}

	if err := ms.notifier.Notify(ctx, message); err != nil {
		err = errors.Wrap(err, "failed to notify recipient")
		slog.Error("failed to notify recipient", "error", err.Error())
		span.SetError(err)
		return err
	}

	return nil
}
