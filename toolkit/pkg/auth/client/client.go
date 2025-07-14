package client

import (
	"context"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	authv1 "github.com/parta4ok/kvs/api/grpc/v1"
)

type AuthClient struct {
	conn   *grpc.ClientConn
	client authv1.AuthServiceClient
}

func New(addr string, opts ...grpc.DialOption) (*AuthClient, error) {
	if len(opts) == 0 {
		opts = []grpc.DialOption{
			grpc.WithTransportCredentials(insecure.NewCredentials()),
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
