package client

import (
	"context"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	authv1 "github.com/parta4ok/kvs/api/grpc/v1"
	"github.com/parta4ok/kvs/toolkit/pkg/tracing/middleware"
)

type AuthClient struct {
	conn   *grpc.ClientConn
	client authv1.AuthServiceClient
}

func New(addr string, opts ...grpc.DialOption) (*AuthClient, error) {
	if len(opts) == 0 {
		opts = []grpc.DialOption{
			grpc.WithTransportCredentials(insecure.NewCredentials()),
			grpc.WithUnaryInterceptor(middleware.UnaryClientInterceptor()),
		}
	}

	conn, err := grpc.NewClient(addr, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to dial auth service: %w", err)
	}

	client := authv1.NewAuthServiceClient(conn)
	return &AuthClient{conn: conn, client: client}, nil
}

func (c *AuthClient) Close() error {
	return c.conn.Close()
}

func (c *AuthClient) Introspect(ctx context.Context, req *authv1.IntrospectRequest,
	opts ...grpc.CallOption) (*authv1.IntrospectResponse, error) {
	return c.client.Introspect(ctx, req, opts...)
}

func (c *AuthClient) GetLinkedUsers(ctx context.Context, req *authv1.LinkedID,
	opts ...grpc.CallOption) (*authv1.LinkedUsersResponse, error) {
	return c.client.GetLinkedUsers(ctx, req, opts...)
}

func (c *AuthClient) GetMentorGroups(ctx context.Context, req *authv1.MentorID,
	opts ...grpc.CallOption) (*authv1.GroupsResponse, error) {
	return c.client.GetMentorGroups(ctx, req, opts...)
}

func (c *AuthClient) GetUserByID(ctx context.Context, req *authv1.UserID,
	opts ...grpc.CallOption) (*authv1.UserInfoResponse, error) {
	return c.client.GetUserByID(ctx, req, opts...)
}

func (c *AuthClient) GetGroupTitleByID(ctx context.Context, req *authv1.GroupID,
	opts ...grpc.CallOption) (*authv1.GroupResponse, error) {
	return c.client.GetGroupTitleByID(ctx, req, opts...)
}
