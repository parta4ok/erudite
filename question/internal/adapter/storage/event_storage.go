package storage

import (
	"context"

	"github.com/parta4ok/kvs/question/internal/entities/event"
)

//go:generate mockgen -source=event_storage.go -destination=./testdata/event_storage.go -package=testdata
type EventStorage interface {
	GetUnpublishedEvents(ctx context.Context) ([]event.Event, error)
	MarkEventAsPublished(ctx context.Context, id int, fn func(ctx context.Context) error) error
	FlushPublishedEvents(ctx context.Context) error
}
