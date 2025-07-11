package private

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"strconv"

	"google.golang.org/grpc"

	authv1 "github.com/parta4ok/kvs/api/grpc/v1"
	"github.com/parta4ok/kvs/auth/internal/entities"
	"github.com/pkg/errors"
)

type AuthService struct {
	authv1.UnimplementedAuthServiceServer
}

type Server struct {
	AuthService
	server  *grpc.Server
	factory CommandFactory
	port    string
}

type ServerOption func(*Server)

func WithFactory(facroty CommandFactory) ServerOption {
	return func(srv *Server) {
		srv.factory = facroty
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
	authService := &AuthService{}
	serv := &Server{
		server:      grpc.NewServer(),
		AuthService: *authService,
	}

	serv.setOptions(opts...)

	if serv.factory == nil {
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

	authv1.RegisterAuthServiceServer(srv.server, &srv.AuthService)

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

func (srv *Server) Introspect(
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

	userIDRaw := req.UserId
	userID, err := strconv.ParseUint(userIDRaw, 10, 32)
	if err != nil {
		err := errors.Wrap(entities.ErrInvalidParam, "extract userID failure")
		slog.Error(err.Error())
		return &authv1.IntrospectResponse{ErrorMessage: err.Error()}, nil
	}

	command, err := srv.factory.NewIntrospectedCommand(ctx, userID, token)
	if err != nil {
		err := errors.Wrap(err, "create introspect command failure")
		slog.Error(err.Error())
		return &authv1.IntrospectResponse{ErrorMessage: err.Error()}, nil
	}

	res, err := command.Exec()
	if err != nil && !res.Success {
		err := errors.Wrap(err, "introspect command exec failure")
		slog.Error(err.Error())
		return &authv1.IntrospectResponse{ErrorMessage: err.Error()}, nil
	}

	return &authv1.IntrospectResponse{}, nil
}
