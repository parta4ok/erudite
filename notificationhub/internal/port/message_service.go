package port

import (
	"context"

	"github.com/parta4ok/kvs/notificationhub/internal/entities"
)

//go:generate mockgen -source=./message_service.go -destination=testdata/message_service.go -package=testdata
type MessageService interface {
	SendMessage(ctx context.Context, message entities.Event) error
}
