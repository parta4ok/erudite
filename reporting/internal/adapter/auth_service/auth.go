package authservice

import (
	"context"
	"log/slog"

	authv1 "github.com/parta4ok/kvs/api/grpc/v1"
	"github.com/parta4ok/kvs/reporting/internal/cases"
	"github.com/parta4ok/kvs/reporting/internal/entities"

	"github.com/parta4ok/kvs/toolkit/pkg/auth/client"
	"github.com/parta4ok/kvs/toolkit/pkg/tracer"
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

	ctx, span, cancel := tracer.Start(ctx, "GetMentorGroups")
	defer cancel()

	req := &authv1.MentorID{
		MentorID: mentorID,
	}

	groups, err := srv.client.GetMentorGroups(ctx, req)
	if err != nil {
		err = errors.Wrapf(entities.ErrInternal, "GetMentorGroups failure: %v", err)
		span.SetError(err)
		slog.Error("GetMentorGroups failure", "error", err.Error())
		return nil, err
	}

	if groups.GetError().GetMessage() != "" {
		err := errors.Wrapf(entities.ErrInternal, "error message: %s", groups.Error.Message)
		span.SetError(err)
		slog.Error("error message", "error", err.Error())
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

func (srv *AuthService) GetLinkedUsers(ctx context.Context, id string,
) (*entities.LinkedMentorAndStudent, error) {
	slog.Info("GetLinkedUsers started", slog.String("id", id))

	ctx, span, cancel := tracer.Start(ctx, "GetMentorGroups")
	defer cancel()

	req := &authv1.LinkedID{
		LinkedID: id,
	}

	linkedUsers, err := srv.client.GetLinkedUsers(ctx, req)
	if err != nil {
		err = errors.Wrapf(entities.ErrInternal, "GetLinkedUsers failure: %v", err)
		span.SetError(err)
		slog.Error("GetLinkedUsers failure", "error", err.Error())
		return nil, err
	}

	if linkedUsers.Error.Message != "" {
		err := errors.Wrapf(entities.ErrInternal, "error message: %s", linkedUsers.Error.Message)
		span.SetError(err)
		slog.Error("error message", "error", err.Error())
		return nil, err
	}

	if linkedUsers.Recipient == nil || linkedUsers.Student == nil {
		err := errors.Wrap(entities.ErrInternal, "nil users info")
		span.SetError(err)
		slog.Error("nil users info", "error", err.Error())
		return nil, err
	}

	student := &entities.User{
		ID:       linkedUsers.Student.Id,
		Name:     linkedUsers.Student.Username,
		Fullname: linkedUsers.Student.Fullname,
		Contacts: linkedUsers.Student.Contacts,
		GroupID:  linkedUsers.Student.GroupId,
	}

	mentor := &entities.User{
		ID:       linkedUsers.Recipient.Id,
		Name:     linkedUsers.Recipient.Username,
		Fullname: linkedUsers.Recipient.Fullname,
		Contacts: linkedUsers.Recipient.Contacts,
		GroupID:  linkedUsers.Recipient.GroupId,
	}

	slog.Info("GetLinkedUsers completed")
	return &entities.LinkedMentorAndStudent{
		Mentor:  mentor,
		Student: student,
	}, nil
}

func (srv *AuthService) GetUserByID(ctx context.Context, userID string) (*entities.User, error) {
	slog.Info("GetGetUserByID started", slog.String("id", userID))

	ctx, span, cancel := tracer.Start(ctx, "GetUserByID")
	defer cancel()

	req := &authv1.UserID{
		UserID: userID,
	}

	user, err := srv.client.GetUserByID(ctx, req)
	if err != nil {
		err = errors.Wrapf(entities.ErrInternal, "GetUserByID failure: %v", err)
		span.SetError(err)
		slog.Error("GetUserByID failure", "error", err.Error())
		return nil, err
	}

	return &entities.User{
		ID:       user.User.Id,
		Name:     user.User.Username,
		Fullname: user.User.Fullname,
		Contacts: user.User.Contacts,
		GroupID:  user.User.GroupId,
	}, nil
}

func (srv *AuthService) GetGroupTitleByID(ctx context.Context, groupID string) (string, error) {
	slog.Info("GetGroupTitleByID started", slog.String("id", groupID))
	ctx, span, cancel := tracer.Start(ctx, "GetGroupTitleByID")
	defer cancel()

	req := &authv1.GroupID{
		GroupID: groupID,
	}

	group, err := srv.client.GetGroupTitleByID(ctx, req)
	if err != nil {
		err = errors.Wrapf(entities.ErrInternal, "GetGroupTitleByID failure: %v", err)
		span.SetError(err)
		slog.Error("GetGroupTitleByID failure", "error", err.Error())
		return "", err
	}

	return group.Group.GetName(), nil
}

func (srv *AuthService) Introspect(ctx context.Context, jwt string) (*entities.Claims, error) {
	slog.Info("Introspect started")

	ctx, span, cancel := tracer.Start(ctx, "Introspect")
	defer cancel()

	req := &authv1.IntrospectRequest{
		Token: jwt,
	}

	resp, err := srv.client.Introspect(ctx, req)
	if err != nil {
		err = errors.Wrapf(entities.ErrInternal, "introspect failure: %v", err)
		span.SetError(err)
		slog.Error("introspect failure", "error", err.Error())
		return nil, err
	}

	if resp.Error.Message != "" {
		err := errors.Wrapf(entities.ErrForbidden, "error message: %s", resp.Error.Message)
		span.SetError(err)
		slog.Error("error message", "error", err.Error())
		return nil, err
	}

	if resp.Claims == nil {
		err := errors.Wrap(entities.ErrForbidden, "nil claims")
		span.SetError(err)
		slog.Error("nil claims", "error", err.Error())
		return nil, err
	}

	slog.Info("Introspect completed")

	return &entities.Claims{
		Username: resp.Claims.Username,
		Subject:  resp.Claims.Subject,
		Rights:   resp.Claims.Rights,
		Audience: resp.Claims.Audience,
		Issuer:   resp.Claims.Issuer,
	}, nil
}
