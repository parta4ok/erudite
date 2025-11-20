package tracing

import (
	"context"

	"github.com/parta4ok/kvs/toolkit/pkg/tracing"
)

// NewJaegerTracerAdapter creates a new Jaeger tracer and returns it as tracing.Tracer interface
func NewJaegerTracerAdapter(serviceName string, endpoint string) (tracing.Tracer, error) {
	jaegerTracer, err := NewJaegerTracer(serviceName, endpoint)
	if err != nil {
		return nil, err
	}
	return &JaegerTracerAdapter{tracer: jaegerTracer}, nil
}

// JaegerTracerAdapter adapts JaegerTracer to implement tracing.Tracer interface
type JaegerTracerAdapter struct {
	tracer *JaegerTracer
}

func (a *JaegerTracerAdapter) Start(ctx context.Context, operationName string) (context.Context, tracing.Span, func()) {
	newCtx, span, cancel := a.tracer.Start(ctx, operationName)
	return newCtx, &JaegerSpanAdapter{span: span.(*JaegerSpan)}, cancel
}

func (a *JaegerTracerAdapter) Close() error {
	return a.tracer.Close()
}

// JaegerSpanAdapter adapts JaegerSpan to implement tracing.Span interface
type JaegerSpanAdapter struct {
	span *JaegerSpan
}

func (a *JaegerSpanAdapter) SetError(err error, message string) {
	a.span.SetError(err, message)
}

func (a *JaegerSpanAdapter) SetTag(key string, value interface{}) {
	a.span.SetTag(key, value)
}

func (a *JaegerSpanAdapter) GetSpanID() string {
	return a.span.GetSpanID()
}

func (a *JaegerSpanAdapter) GetTraceID() string {
	return a.span.GetTraceID()
}

func (a *JaegerSpanAdapter) Start(ctx context.Context) error {
	return a.span.Start(ctx)
}

func (a *JaegerSpanAdapter) Stop(ctx context.Context) error {
	return a.span.Stop(ctx)
}

func (a *JaegerSpanAdapter) Type() string {
	return a.span.Type()
}
