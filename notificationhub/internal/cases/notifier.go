package cases

import "github.com/parta4ok/kvs/notificationhub/internal/entities"

//go:generate mockgen -source=notifier.go -destination=testdata/notifier.go -package=testdata
type Notifier interface {
	Notify(sessionResult *entities.SessionResult, linkedUsers *entities.LinkedUsers) error
	Next() Notifier
	SetNextNotifier(notifier Notifier)
}
