package cases

import (
	"context"

	"github.com/parta4ok/kvs/reporting/internal/entities"
)

//go:generate mockgen -source=./question_client.go -destination=./testdata/question_client.go -package=testdata
type QuestionClient interface {
	GetPassedStudentsTopics(ctx context.Context, students []string) (
		map[string][]entities.Topic, error)
}
