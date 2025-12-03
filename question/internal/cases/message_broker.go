package cases

import (
	"context"

	"github.com/parta4ok/kvs/question/internal/entities/event"
)

//go:generate mockgen -source=./message_broker.go -destination=./testdata/message_broker.go -package=testdata
type MessageBroker interface {
	Publish(ctx context.Context, event event.Event) error
}
