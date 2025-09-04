package tracing

import (
	"context"
	"io"

	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/parta4ok/kvs/toolkit/pkg/tracing"
	"github.com/uber/jaeger-client-go"
	"github.com/uber/jaeger-client-go/config"
)

var (
	_ tracing.Tracer = (*JaegerTracer)(nil)
	_ tracing.Span   = (*JaegerSpan)(nil)
)

type JaegerTracer struct {
	tracer opentracing.Tracer
	closer io.Closer
}

type JaegerSpan struct {
	span opentracing.Span
}

func NewJaegerTracer(serviceName string, jaegerEndpoint string) (*JaegerTracer, error) {
	cfg := config.Configuration{
		ServiceName: serviceName,
		Sampler: &config.SamplerConfig{
			Type:  jaeger.SamplerTypeConst,
			Param: 1,
		},
		Reporter: &config.ReporterConfig{
			LogSpans:            false,
			BufferFlushInterval: 1000,
			LocalAgentHostPort:  jaegerEndpoint,
		},
	}

	tracer, closer, err := cfg.NewTracer()
	if err != nil {
		return nil, err
	}

	opentracing.SetGlobalTracer(tracer)

	return &JaegerTracer{
		tracer: tracer,
		closer: closer,
	}, nil
}

func (jt *JaegerTracer) Start(ctx context.Context, operationName string) (context.Context,
	tracing.Span, func()) {
	var span opentracing.Span

	if parentSpan := opentracing.SpanFromContext(ctx); parentSpan != nil {
		span = jt.tracer.StartSpan(operationName, opentracing.ChildOf(parentSpan.Context()))
	} else {
		span = jt.tracer.StartSpan(operationName)
	}

	newCtx := opentracing.ContextWithSpan(ctx, span)

	cancel := func() {
		span.Finish()
	}

	return newCtx, &JaegerSpan{span: span}, cancel
}

func (jt *JaegerTracer) Close() error {
	return jt.closer.Close()
}

func (js *JaegerSpan) SetError(err error, message string) {
	ext.Error.Set(js.span, true)
	js.span.SetTag("error.message", message)
	if err != nil {
		js.span.SetTag("error.object", err.Error())
	}
}

func (js *JaegerSpan) SetTag(key string, value interface{}) {
	js.span.SetTag(key, value)
}

func (js *JaegerSpan) GetSpanID() string {
	return js.span.Context().(jaeger.SpanContext).SpanID().String()
}

func (js *JaegerSpan) GetTraceID() string {
	return js.span.Context().(jaeger.SpanContext).TraceID().String()
}
