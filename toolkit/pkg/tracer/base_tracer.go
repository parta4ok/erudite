// toolkit/pkg/tracer/base_tracer.go
package tracer

import (
	"context"
	"sync/atomic"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"

	"github.com/parta4ok/kvs/toolkit/pkg/port"
)

const portType = "tracer"

var (
	active     atomic.Bool
	tracerName atomic.Value
)

var _ port.BasePort = (*Port)(nil)

type Port struct {
	serviceName    string
	serviceVersion string
	endpoint       string
	enabled        bool

	shutdown func(context.Context) error
}

func NewPort(serviceName, serviceVersion, endpoint string, enabled bool) *Port {
	return &Port{
		serviceName:    serviceName,
		serviceVersion: serviceVersion,
		endpoint:       endpoint,
		enabled:        enabled,
	}
}

func (p *Port) Start(ctx context.Context) error {
	if !p.enabled {
		return nil
	}

	exporter, err := otlptracehttp.New(ctx,
		otlptracehttp.WithEndpoint(p.endpoint),
		otlptracehttp.WithInsecure(),
	)
	if err != nil {
		return err //nolint:wrapcheck //ok
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(p.serviceName),
			semconv.ServiceVersion(p.serviceVersion),
		),
	)
	if err != nil {
		return err //nolint:wrapcheck //ok
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	tracerName.Store(p.serviceName)
	p.shutdown = tp.Shutdown
	active.Store(true)

	return nil
}

func (p *Port) Stop(ctx context.Context) error {
	active.Store(false)
	if p.shutdown == nil {
		return nil
	}
	return p.shutdown(ctx) //nolint:wrapcheck //ok
}

func (p *Port) Type() string {
	return portType
}

type Span struct {
	otelSpan trace.Span
}

func (s *Span) SetError(err error) {
	if err == nil {
		return
	}
	s.otelSpan.RecordError(err)
	s.otelSpan.SetStatus(codes.Error, err.Error())
}

func Start(ctx context.Context, spanName string) (context.Context, *Span, func()) {
	if !active.Load() {
		return ctx, &Span{otelSpan: trace.SpanFromContext(ctx)}, func() {}
	}
	instrumentationName, _ := tracerName.Load().(string)
	ctx, otelSpan := otel.Tracer(instrumentationName).Start(ctx, spanName)
	return ctx, &Span{otelSpan: otelSpan}, func() { otelSpan.End() }
}
