package questionservice

import (
	"context"
	"log/slog"

	"github.com/parta4ok/kvs/reporting/internal/cases"
	"github.com/parta4ok/kvs/reporting/internal/entities"
	"github.com/parta4ok/kvs/toolkit/pkg/question/client"
	"github.com/parta4ok/kvs/toolkit/pkg/tracer"
	"github.com/pkg/errors"
)

var (
	_ cases.QuestionClient = (*QuestionClient)(nil)
)

type QuestionClient struct {
	client *client.Client
}

func New(port string) (*QuestionClient, error) {
	if port == "" {
		return nil, errors.Wrap(entities.ErrInvalidParam, "question service port not set")
	}

	client, err := client.New(port)
	if err != nil {
		return nil, errors.Wrap(err, "new client for question service failure")
	}

	return &QuestionClient{
		client: client,
	}, nil
}

func (q *QuestionClient) GetPassedStudentsTopics(ctx context.Context, students []string) (
	map[string][]entities.Topic, error) {
	slog.Info("GetPassedStudentsTopics started")
	ctx, span, cancel := tracer.Start(ctx, "GetPassedStudentsTopicsSpan")
	defer cancel()

	passedTopics, err := q.client.GetPassedStudentsTopics(ctx, students)
	if err != nil {
		if errors.Is(err, client.ErrBadRequest) {
			err = errors.Wrapf(entities.ErrInvalidParam, "GetPassedStudentsTopics failure: %v", err)
			slog.Error("GetPassedStudentsTopics failure", "error", err)
			span.SetError(err)
			return nil, err
		}
		err = errors.Wrapf(entities.ErrInternal, "GetPassedStudentsTopics failure: %v", err)
		slog.Error("GetPassedStudentsTopics failure", "error", err)
		span.SetError(err)
		return nil, err
	}

	res := make(map[string][]entities.Topic)
	for student, topics := range passedTopics.StudentsTopics {
		if _, ok := res[student]; !ok {
			res[student] = make([]entities.Topic, 0, len(topics))
		}
		for _, topic := range topics {
			res[student] = append(res[student], entities.Topic{
				ID:    topic.ID,
				Title: topic.Title,
			})
		}
	}

	slog.Info("GetPassedStudentsTopics completed")
	return res, nil
}
