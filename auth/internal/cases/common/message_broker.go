package common

import (
	"context"

	"github.com/parta4ok/kvs/auth/internal/entities"
)

//go:generate mockgen -source=message_broker.go -destination=./testdata/message_broker.go -package=testdata
type MessageBroker interface {
	SendEvent(ctx context.Context, event entities.Event) error
}
