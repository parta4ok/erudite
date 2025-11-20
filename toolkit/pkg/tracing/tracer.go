package tracing

import (
	"context"
)

type NoOpTracer struct{}

func (n *NoOpTracer) Start(ctx context.Context, operationName string) (
	context.Context, Span, func()) {
	return ctx, &NoOpSpan{}, func() {}
}

func (n *NoOpTracer) Close() error {
	return nil
}

type NoOpSpan struct{}

func (n *NoOpSpan) SetError(err error, message string) {}

func (n *NoOpSpan) SetTag(key string, value interface{}) {}

func (n *NoOpSpan) GetSpanID() string {
	return ""
}

func (n *NoOpSpan) GetTraceID() string {
	return ""
}

func (t *NoOpSpan) Start(_ context.Context) error {
	return nil
}

func (t *NoOpSpan) Stop(_ context.Context) error {
	return nil
}

func (t *NoOpSpan) Type() string {
	return noOpType
}
