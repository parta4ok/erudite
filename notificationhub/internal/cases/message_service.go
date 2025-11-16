package cases

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/parta4ok/kvs/notificationhub/internal/entities"
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
	if message == nil {
		err := errors.Wrap(entities.ErrInvalidParam, "message is nil")
		slog.Error(err.Error())
		return err
	}

	fmt.Println(message.GetRecipient())

	if err := ms.notifier.Notify(ctx, message); err != nil {
		err = errors.Wrap(err, "failed to notify recipient")
		slog.Error(err.Error())
		return err
	}

	return nil
}
