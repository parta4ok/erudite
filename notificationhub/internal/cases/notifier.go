package cases

import (
	"context"

	"github.com/parta4ok/kvs/notificationhub/internal/entities"
)

//go:generate mockgen -source=notifier.go -destination=testdata/notifier.go -package=testdata
type Notifier interface {
	Notify(ctx context.Context, message entities.Event) error
	Next() Notifier
	SetNextNotifier(notifier Notifier)
}
