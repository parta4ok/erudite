package port

import (
	"context"

	"github.com/parta4ok/kvs/reporting/internal/entities"
)

//go:generate mockgen -source=service.go -destination=./testdata/service.go -package=testdata
type Service interface {
	DeliverySessionResult(ctx context.Context, session *entities.SessionResult) error
	GetPassedTopicsByGroups(ctx context.Context, mentorID string) error
}
