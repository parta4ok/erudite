package cases

import (
	"log/slog"

	"github.com/parta4ok/kvs/notificationhub/internal/entities"
	"github.com/pkg/errors"
)

type MessageService struct {
	notifier   Notifier
	authClient AuthClient
}

func NewMessageService(notifier Notifier, authClient AuthClient) (*MessageService, error) {
	if notifier == nil {
		return nil, errors.Wrap(entities.ErrInvalidParam, "notifier is nil")
	}
	if authClient == nil {
		return nil, errors.Wrap(entities.ErrInvalidParam, "authClient is nil")
	}

	return &MessageService{
		notifier:   notifier,
		authClient: authClient,
	}, nil
}

func (ms *MessageService) SendMessage(sessionResult *entities.SessionResult) error {
	if sessionResult == nil {
		err := errors.Wrap(entities.ErrInvalidParam, "sessionResult is nil")
		slog.Error(err.Error())
		return err
	}

	recipient, err := ms.authClient.GetRecipientByID(sessionResult.GetUserID())
	if err != nil {
		err = errors.Wrap(err, "failed to get recipient")
		slog.Error(err.Error())
		return err
	}

	if err := ms.notifier.Notify(sessionResult, recipient); err != nil {
		err = errors.Wrap(err, "failed to notify recipient")
		slog.Error(err.Error())
		return err
	}

	return nil
}
