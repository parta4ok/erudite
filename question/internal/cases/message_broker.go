package cases

import (
	"context"

	"github.com/parta4ok/kvs/question/internal/entities"
)

//go:generate mockgen -source=./message_broker.go -destination=./testdata/message_broker.go -package=testdata
type MessageBroker interface {
	SessionFinishedEvent(ctx context.Context, sessionResult *entities.SessionResult) error
}
