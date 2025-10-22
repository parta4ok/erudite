package authservice

import (
	"context"
	"log/slog"

	authv1 "github.com/parta4ok/kvs/api/grpc/v1"
	"github.com/parta4ok/kvs/reporting/internal/cases"
	"github.com/parta4ok/kvs/reporting/internal/entities"

	"github.com/parta4ok/kvs/toolkit/pkg/auth/client"
	"github.com/pkg/errors"
)

var (
	_ cases.AuthClient = (*AuthService)(nil)
)

type AuthService struct {
	client *client.AuthClient
}

func NewAuthService(port string) (*AuthService, error) {
	if port == "" {
		err := errors.Wrap(entities.ErrInvalidParam, "port not set")
		return nil, err
	}

	c, err := client.New(port)
	if err != nil {
		err = errors.Wrap(entities.ErrInternal, "creating auth grpc client failure")
		return nil, err
	}

	return &AuthService{client: c}, nil
}

func (srv *AuthService) GetMentorGroups(ctx context.Context, mentorID string) (
	[]entities.Student, error) {
	slog.Info("GetMentorGroups started", slog.String("mentor_id", mentorID))

	req := &authv1.MentorID{
		MentorID: mentorID,
	}

	groups, err := srv.client.GetMentorGroups(ctx, req)
	if err != nil {
		err = errors.Wrapf(entities.ErrInternal, "GetMentorGroups failure: %v", err)
		slog.Error(err.Error())
		return nil, err
	}

	if groups.GetError().GetMessage() != "" {
		err := errors.Wrapf(entities.ErrInternal, "error message: %s", groups.Error.Message)
		slog.Error(err.Error())
		return nil, err
	}

	slog.Info("GetMentorGroups completed")

	var result = make([]entities.Student, 0, len(groups.GetGroups()))

	for _, group := range groups.GetGroups() {
		for _, student := range group.GetStudents() {
			result = append(result, entities.Student{
				ID:       student.GetId(),
				Name:     student.GetName(),
				FullName: student.GetFullname(),
				Group: entities.Group{
					ID:    group.GetId(),
					Title: group.GetName(),
				},
			})
		}
	}

	return result, nil
}
