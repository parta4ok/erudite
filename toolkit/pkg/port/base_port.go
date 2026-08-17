package port

import "context"

type BasePort interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
}
