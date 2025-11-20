package tracing

import (
	"context"
)

type Tracer interface {
	Start(ctx context.Context, operationName string) (context.Context, Span, func())
	Close() error
}

type Span interface {
	SetError(err error, message string)
	SetTag(key string, value interface{})
	GetSpanID() string
	GetTraceID() string
	Start(context.Context) error
	Stop(context.Context) error
	Type() string
}
