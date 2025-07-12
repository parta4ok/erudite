package entities

import "context"

//go:generate mockgen -source=session_storage.go -destination=./testdata/session_storage.go -package=testdata
type SessionStorage interface {
	IsDailySessionLimitReached(ctx context.Context, userID uint64, topics []string) (bool, error)
}
