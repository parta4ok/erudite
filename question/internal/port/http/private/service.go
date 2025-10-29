package private

import (
	"context"

	"github.com/parta4ok/kvs/question/internal/entities"
)

//go:generate mockgen -source=service.go -destination=./testdata/service.go -package=testdata
type Service interface {
	GetPassedStudentsTopics(ctx context.Context, students []string) (
		map[string][]*entities.Topic, error)
}
