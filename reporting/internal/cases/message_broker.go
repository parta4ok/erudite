package cases

import (
	"context"

	"github.com/parta4ok/kvs/reporting/internal/entities"
)

//go:generate mockgen -source=./message_broker.go -destination=./testdata/message_broker.go -package=testdata
type MessageBroker interface {
	ReportEvent(ctx context.Context, sessionResult *entities.PassedTopicsReport,
		representer Representer) error
}
