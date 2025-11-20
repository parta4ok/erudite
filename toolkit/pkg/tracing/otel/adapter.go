package otel

import (
	"context"

	"github.com/parta4ok/kvs/toolkit/pkg/tracing"
)

// NewOtelTracerAdapter creates a new OTEL tracer and returns it as tracing.Tracer interface
func NewOtelTracerAdapter(serviceName string, endpoint string) (tracing.Tracer, error) {
	otelTracer, err := NewOtelTracer(serviceName, endpoint)
	if err != nil {
		return nil, err
	}
	return &OtelTracerAdapter{tracer: otelTracer}, nil
}

// OtelTracerAdapter adapts OtelTracer to implement tracing.Tracer interface
type OtelTracerAdapter struct {
	tracer *OtelTracer
}

func (a *OtelTracerAdapter) Start(ctx context.Context, operationName string) (context.Context, tracing.Span, func()) {
	newCtx, span, cancel := a.tracer.Start(ctx, operationName)
	return newCtx, &OtelSpanAdapter{span: span.(*OtelSpan)}, cancel
}

func (a *OtelTracerAdapter) Close() error {
	return a.tracer.Close()
}

// OtelSpanAdapter adapts OtelSpan to implement tracing.Span interface
type OtelSpanAdapter struct {
	span *OtelSpan
}

func (a *OtelSpanAdapter) SetError(err error, message string) {
	a.span.SetError(err, message)
}

func (a *OtelSpanAdapter) SetTag(key string, value interface{}) {
	a.span.SetTag(key, value)
}

func (a *OtelSpanAdapter) GetSpanID() string {
	return a.span.GetSpanID()
}

func (a *OtelSpanAdapter) GetTraceID() string {
	return a.span.GetTraceID()
}

func (a *OtelSpanAdapter) Start(ctx context.Context) error {
	return a.span.Start(ctx)
}

func (a *OtelSpanAdapter) Stop(ctx context.Context) error {
	return a.span.Stop(ctx)
}

func (a *OtelSpanAdapter) Type() string {
	return a.span.Type()
}
