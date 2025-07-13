package private

import (
	"context"
	"fmt"
	"log/slog"
	"net"

	"google.golang.org/grpc"

	authv1 "github.com/parta4ok/kvs/api/grpc/v1"
	"github.com/parta4ok/kvs/auth/internal/entities"
	"github.com/pkg/errors"
)

type AuthService struct {
	authv1.UnimplementedAuthServiceServer
	factory CommandFactory
}

func (a *AuthService) Introspect(
	ctx context.Context,
	req *authv1.IntrospectRequest,
) (*authv1.IntrospectResponse, error) {
	slog.Info("Introspect started")

	token := req.Token
	if token == "" {
		err := errors.Wrap(entities.ErrInvalidJWT, "jwt token is empty")
		slog.Error(err.Error())
		return &authv1.IntrospectResponse{ErrorMessage: err.Error()}, nil
	}

	if req.UserId == "" {
		err := errors.Wrap(entities.ErrInvalidParam, "extract userID failure")
		slog.Error(err.Error())
		return &authv1.IntrospectResponse{ErrorMessage: err.Error()}, nil
	}
	slog.Info("------", slog.String("jwt", token), slog.String("userID", req.UserId))
	command, err := a.factory.NewIntrospectedCommand(ctx, req.UserId, token)
	if err != nil {
		err := errors.Wrap(err, "create introspect command failure")
		slog.Error(err.Error())
		return &authv1.IntrospectResponse{ErrorMessage: err.Error()}, nil
	}

	res, err := command.Exec()
	if err != nil {
		err := errors.Wrap(err, "introspect command exec failure")
		slog.Error(err.Error())
		return &authv1.IntrospectResponse{ErrorMessage: err.Error()}, nil
	}

	if res != nil {
		if !res.Success {
			err := errors.Wrap(entities.ErrInvalidJWT, "introspect command exec failure")
			slog.Error(err.Error())
			return &authv1.IntrospectResponse{ErrorMessage: err.Error()}, nil
		}
	}

	return &authv1.IntrospectResponse{}, nil
}

type Server struct {
	authService *AuthService
	server      *grpc.Server
	port        string
}

type ServerOption func(*Server)

func WithFactory(factory CommandFactory) ServerOption {
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
		server:      grpc.NewServer(),
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

func (srv *Server) Stop() {
	slog.Info("stop gRPC server")
	srv.server.Stop()
}
