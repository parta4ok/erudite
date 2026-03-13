package private

import (
	"context"
	"fmt"
	"log/slog"
	"net"

	"google.golang.org/grpc"

	authv1 "github.com/parta4ok/kvs/api/grpc/v1"
	"github.com/parta4ok/kvs/auth/internal/entities"
	"github.com/parta4ok/kvs/auth/internal/port"
	"github.com/parta4ok/kvs/toolkit/pkg/tracing"
	"github.com/parta4ok/kvs/toolkit/pkg/tracing/middleware"
	"github.com/pkg/errors"
)

const grpcPortType = "grpc_private"

type AuthService struct {
	authv1.UnimplementedAuthServiceServer
	factory port.CommandFactory
}

//nolint:funlen,dupl //ok
func (a *AuthService) Introspect(ctx context.Context, req *authv1.IntrospectRequest,
) (*authv1.IntrospectResponse, error) {
	slog.Info("Introspect started")
	ctx, span, cancel := tracing.GlobalTracer().Start(ctx, "IntrospectGRPCHandlerSpan")
	defer cancel()

	token := req.Token
	if token == "" {
		err := errors.Wrap(entities.ErrInvalidJWT, "jwt token is empty")
		slog.Error(err.Error())
		span.SetError(err, "token is empty")
		return &authv1.IntrospectResponse{
			Claims: nil,
			Error:  &authv1.Error{Message: err.Error()},
		}, nil
	}

	res, err := a.factory.NewIntrospectedCommand(token).Exec(ctx)
	if err != nil {
		err := errors.Wrap(err, "introspect command exec failure")
		slog.Error(err.Error())
		span.SetError(err, "introspect command exec failure")
		return &authv1.IntrospectResponse{
			Claims: nil,
			Error:  &authv1.Error{Message: err.Error()},
		}, nil
	}

	if res != nil {
		if !res.Success {
			err := errors.Wrap(entities.ErrInvalidJWT, "introspect command exec failure")
			slog.Error(err.Error())
			span.SetError(err, "introspect command exec failure")
			return &authv1.IntrospectResponse{
				Claims: nil,
				Error:  &authv1.Error{Message: err.Error()},
			}, nil
		}
	}

	userClaims, ok := res.Payload.(*entities.UserClaims)
	if !ok {
		err := errors.Wrap(entities.ErrInvalidJWT, "assert claims failure")
		slog.Error(err.Error())
		span.SetError(err, "assert claims failure")
		return &authv1.IntrospectResponse{
			Claims: nil,
			Error:  &authv1.Error{Message: err.Error()},
		}, nil
	}

	resp := &authv1.IntrospectResponse{
		Claims: &authv1.UserClaims{
			Username: userClaims.Username,
			Issuer:   userClaims.Issuer,
			Audience: userClaims.Audience,
			Subject:  userClaims.Subject,
			Rights:   userClaims.Rights,
		},
		Error: &authv1.Error{Message: ""},
	}

	return resp, nil
}

//nolint:funlen,dupl //ok
func (a *AuthService) GetLinkedUsers(ctx context.Context, req *authv1.LinkedID,
) (*authv1.LinkedUsersResponse, error) {
	slog.Info("GetLinkedUser started")
	ctx, span, cancel := tracing.GlobalTracer().Start(ctx, "GetLinkedUsersGRPCHandlerSpan")
	defer cancel()

	userID := req.LinkedID
	if userID == "" {
		err := errors.Wrap(entities.ErrInvalidParam, "user id is empty")
		slog.Error(err.Error())
		span.SetError(err, "user id is empty")
		return &authv1.LinkedUsersResponse{
			Recipient: nil,
			Student:   nil,
			Error:     &authv1.Error{Message: err.Error()},
		}, nil
	}

	res, err := a.factory.NewGetLinkedUsersCommand(userID).Exec(ctx)
	if err != nil {
		err := errors.Wrap(err, "get user command exec failure")
		slog.Error(err.Error())
		span.SetError(err, "get user command exec failure")
		return &authv1.LinkedUsersResponse{
			Recipient: nil,
			Student:   nil,
			Error:     &authv1.Error{Message: err.Error()},
		}, nil
	}

	if res != nil {
		if !res.Success {
			err := errors.Wrap(entities.ErrInvalidJWT, "get user command exec failure")
			slog.Error(err.Error())
			span.SetError(err, "get user command exec failure")
			return &authv1.LinkedUsersResponse{
				Recipient: nil,
				Student:   nil,
				Error:     &authv1.Error{Message: err.Error()},
			}, nil
		}
	}

	userData, ok := res.Payload.(*entities.LinkedUsers)
	if !ok {
		err := errors.Wrap(entities.ErrInvalidJWT, "assert linked users data failure")
		slog.Error(err.Error())
		span.SetError(err, "assert linked users data failure")
		return &authv1.LinkedUsersResponse{
			Recipient: nil,
			Student:   nil,
			Error:     &authv1.Error{Message: err.Error()},
		}, nil
	}

	resp := &authv1.LinkedUsersResponse{
		Recipient: &authv1.UserInfo{
			Id:       userData.Recipient.ID,
			Username: userData.Recipient.Username,
			Fullname: userData.Recipient.FullName,
			Rights:   userData.Recipient.Rights,
			Contacts: userData.Recipient.Contacts,
			GroupId:  userData.Recipient.GroupID,
		},
		Student: &authv1.UserInfo{
			Id:       userData.Student.ID,
			Username: userData.Student.Username,
			Fullname: userData.Student.FullName,
			Rights:   userData.Student.Rights,
			Contacts: userData.Student.Contacts,
			GroupId:  userData.Student.GroupID,
		},
		Error: &authv1.Error{
			Message: "",
		},
	}

	return resp, nil
}

//nolint:funlen //ok
func (a *AuthService) GetMentorGroups(ctx context.Context, req *authv1.MentorID,
) (*authv1.GroupsResponse, error) {
	slog.Info("GetLinkedUser started")
	ctx, span, cancel := tracing.GlobalTracer().Start(ctx, "GetMentorGroupsGRPCHandlerSpan")
	defer cancel()

	if req.GetMentorID() == "" {
		err := errors.Wrap(entities.ErrInvalidParam, "mentorID is invalid")
		slog.Error(err.Error())
		span.SetError(err, "mentorID is invalid")
		return &authv1.GroupsResponse{
			Groups: nil,
			Error:  &authv1.Error{Message: err.Error()},
		}, nil
	}

	res, err := a.factory.NewGetMentorGroupsCommand(req.GetMentorID()).Exec(ctx)
	if err != nil {
		err := errors.Wrap(err, "GetMentorGroupsCommand execution failed")
		slog.Error(err.Error())
		span.SetError(err, "GetMentorGroupsCommand execution failed")
		return &authv1.GroupsResponse{
			Groups: nil,
			Error:  &authv1.Error{Message: err.Error()},
		}, nil
	}

	groups, ok := res.Payload.([]*entities.Group)
	if !ok {
		err := errors.Wrap(entities.ErrInternal, "cast command result failure")
		slog.Error(err.Error())
		span.SetError(err, "cast command result failure")
		return &authv1.GroupsResponse{
			Groups: nil,
			Error:  &authv1.Error{Message: err.Error()},
		}, nil
	}

	groupsResp := &authv1.GroupsResponse{}
	for _, group := range groups {
		var (
			groupResp = &authv1.Group{}
		)
		groupResp.Id = group.GetID()
		groupResp.Name = group.GetName()
		groupResp.Students = make([]*authv1.Student, 0, len(group.GetStudents()))
		for _, student := range group.GetStudents() {
			var (
				studentResp = &authv1.Student{}
			)
			studentResp.Id = student.GetID()
			studentResp.Name = student.GetName()
			studentResp.Fullname = student.GetFullname()

			groupResp.Students = append(groupResp.Students, studentResp)
		}
		groupsResp.Groups = append(groupsResp.Groups, groupResp)
	}

	return groupsResp, nil
}

//nolint:funlen //ok
func (a *AuthService) GetUserByID(ctx context.Context, req *authv1.UserID,
) (*authv1.UserInfoResponse, error) {
	slog.Info("GetUserByID started")
	ctx, span, cancel := tracing.GlobalTracer().Start(ctx, "GetUserByIDGRPCHandlerSpan")
	defer cancel()

	if req.GetUserID() == "" {
		err := errors.Wrap(entities.ErrInvalidParam, "user id is invalid")
		slog.Error(err.Error())
		span.SetError(err, "user id is invalid")
		return &authv1.UserInfoResponse{
			User:  nil,
			Error: &authv1.Error{Message: err.Error()},
		}, nil
	}

	res, err := a.factory.NewGetUserByIDCommand(req.GetUserID()).Exec(ctx)
	if err != nil {
		err := errors.Wrap(err, "GetUserByIDCommand execution failed")
		slog.Error(err.Error())
		span.SetError(err, "GetUserByIDCommand execution failed")
		return &authv1.UserInfoResponse{
			User:  nil,
			Error: &authv1.Error{Message: err.Error()},
		}, nil
	}

	user, ok := res.Payload.(*entities.User)
	if !ok {
		err := errors.Wrap(entities.ErrInternal, "cast command result failure")
		slog.Error(err.Error())
		span.SetError(err, "cast command result failure")
		return &authv1.UserInfoResponse{
			User:  nil,
			Error: &authv1.Error{Message: err.Error()},
		}, nil
	}

	userResponse := &authv1.UserInfoResponse{
		User: &authv1.UserInfo{
			Id:       user.ID,
			Username: user.Username,
			Fullname: user.FullName,
			Rights:   user.Rights,
			Contacts: user.Contacts,
			GroupId:  user.GroupID,
		},
		Error: nil,
	}

	return userResponse, nil
}

//nolint:funlen //ok
func (a *AuthService) GetGroupTitleByID(ctx context.Context, req *authv1.GroupID,
) (*authv1.GroupResponse, error) {
	slog.Info("GetGroupTitleByID started")
	ctx, span, cancel := tracing.GlobalTracer().Start(ctx, "GetGroupTitleByIDGRPCHandlerSpan")
	defer cancel()

	if req.GetGroupID() == "" {
		err := errors.Wrap(entities.ErrInvalidParam, "group id is invalid")
		slog.Error(err.Error())
		span.SetError(err, "group id is invalid")
		return &authv1.GroupResponse{
			Group: nil,
			Error: &authv1.Error{Message: err.Error()},
		}, nil
	}

	res, err := a.factory.NewGetGroupTitleByIDCommand(req.GetGroupID()).Exec(ctx)
	if err != nil {
		err := errors.Wrap(err, "GetGroupTitleByIDCommand execution failed")
		slog.Error(err.Error())
		span.SetError(err, "GetGroupTitleByIDCommand execution failed")
		return &authv1.GroupResponse{
			Group: nil,
			Error: &authv1.Error{Message: err.Error()},
		}, nil
	}

	title, ok := res.Payload.(string)
	if !ok {
		err := errors.Wrap(entities.ErrInternal, "cast command result failure")
		slog.Error(err.Error())
		span.SetError(err, "cast command result failure")
		return &authv1.GroupResponse{
			Group: nil,
			Error: &authv1.Error{Message: err.Error()},
		}, nil
	}

	groupResponse := &authv1.GroupResponse{
		Group: &authv1.Group{
			Id:   req.GetGroupID(),
			Name: title,
		},
		Error: nil,
	}

	return groupResponse, nil
}

type Server struct {
	authService *AuthService
	server      *grpc.Server
	port        string
	listener    net.Listener
}

type ServerOption func(*Server)

func WithFactory(factory port.CommandFactory) ServerOption {
	return func(srv *Server) {
		srv.authService.factory = factory
	}
}

func WithPort(port string) ServerOption {
	return func(srv *Server) {
		srv.port = port
	}
}

func (srv *Server) setOptions(opts ...ServerOption) {
	for _, opt := range opts {
		opt(srv)
	}
}

func NewServer(opts ...ServerOption) (*Server, error) {
	serv := &Server{
		server: grpc.NewServer(
			grpc.UnaryInterceptor(middleware.UnaryServerInterceptor()),
		),
		authService: &AuthService{},
	}

	serv.setOptions(opts...)

	if serv.authService.factory == nil {
		return nil, errors.Wrap(entities.ErrInvalidParam, "factory not set")
	}

	if serv.port == "" {
		return nil, errors.Wrap(entities.ErrInvalidParam, "port not set")
	}

	return serv, nil
}

//nolint:gosec //ok
func (srv *Server) Start(ctx context.Context) error {
	slog.Info("gRPC server starting", "port", srv.port)

	listener, err := net.Listen("tcp", fmt.Sprintf(":%s", srv.port))
	if err != nil {
		return errors.Wrapf(entities.ErrInternal, "net listen failure: %v", err)
	}

	srv.listener = listener
	authv1.RegisterAuthServiceServer(srv.server, srv.authService)

	go func() {
		if err := srv.server.Serve(listener); err != nil {
			slog.Error("gRPC server serve error", "error", err)
		}
	}()

	slog.Info("gRPC server started", "port", srv.port)
	return nil
}

func (srv *Server) Stop(ctx context.Context) error {
	slog.Info("Stopping gRPC server", "port", srv.port)

	if srv.server != nil {
		srv.server.GracefulStop()
	}

	if srv.listener != nil {
		srv.listener.Close() //nolint:errcheck,gosec //ok
	}

	slog.Info("gRPC server stopped", "port", srv.port)
	return nil
}

func (srv *Server) Type() string {
	return grpcPortType
}

// Legacy methods for backward compatibility
func (srv *Server) StartServer() {
	slog.Info("gRPC server started")

	listner, err := net.Listen("tcp", fmt.Sprintf(":%s", srv.port))
	if err != nil {
		err := errors.Wrapf(entities.ErrInternal, "net listen failure: %v", err)
		slog.Error(err.Error())
		return
	}

	authv1.RegisterAuthServiceServer(srv.server, srv.authService)

	if err := srv.server.Serve(listner); err != nil {
		err := errors.Wrapf(entities.ErrInternal, "serve failure: %v", err)
		slog.Error(err.Error())
		return
	}

	slog.Info("gRPC server stopped")
}

func (srv *Server) LegacyStop() {
	slog.Info("stop gRPC server")
	srv.server.Stop()
}
