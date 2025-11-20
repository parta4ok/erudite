package otel

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/parta4ok/kvs/toolkit/pkg/tracing"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	oteltrace "go.opentelemetry.io/otel/trace"
)

var (
	_ tracing.Tracer = (*OtelTracer)(nil)
	_ tracing.Span   = (*OtelSpan)(nil)
)

type OtelTracer struct {
	tracer   oteltrace.Tracer
	provider *sdktrace.TracerProvider
	exporter sdktrace.SpanExporter
}

type OtelSpan struct {
	span oteltrace.Span
}

//nolint:funlen,gocritic //ok
func NewOtelTracer(serviceName string, endpoint string) (*OtelTracer, error) {
	slog.Info("Creating OTLP exporter", "endpoint", endpoint, "service", serviceName)

	var exporter sdktrace.SpanExporter
	var err error

	if strings.HasPrefix(endpoint, "http://") || strings.HasPrefix(endpoint, "https://") {
		slog.Info("Using OTLP HTTP protocol", "endpoint", endpoint)
		exporter, err = otlptracehttp.New(
			context.Background(),
			otlptracehttp.WithEndpoint(endpoint),
			otlptracehttp.WithURLPath("/v1/traces"),
			otlptracehttp.WithInsecure(),
		)
	} else if strings.Contains(endpoint, ":4318") {
		httpEndpoint := "http://" + endpoint
		slog.Info("Using OTLP HTTP protocol (auto-detected)", "endpoint", httpEndpoint)
		exporter, err = otlptracehttp.New(
			context.Background(),
			otlptracehttp.WithEndpoint(httpEndpoint),
			otlptracehttp.WithURLPath("/v1/traces"),
			otlptracehttp.WithInsecure(),
		)
	} else {
		slog.Info("Using OTLP gRPC protocol", "endpoint", endpoint)
		exporter, err = otlptracegrpc.New(
			context.Background(),
			otlptracegrpc.WithEndpoint(endpoint),
			otlptracegrpc.WithInsecure(),
		)
	}

	if err != nil {
		slog.Error("Failed to create OTLP exporter", "error", err, "endpoint", endpoint)
		return nil, fmt.Errorf("failed to create OTLP exporter: %w", err)
	}
	slog.Info("OTLP exporter created successfully", "endpoint", endpoint)

	resource := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceName(serviceName),
		semconv.ServiceVersion("1.0.0"),
	)

	// Create trace provider with simple span processor for immediate flush
	slog.Info("Creating TracerProvider with SimpleSpanProcessor", "service", serviceName)
	provider := sdktrace.NewTracerProvider(
		sdktrace.WithSpanProcessor(sdktrace.NewSimpleSpanProcessor(exporter)),
		sdktrace.WithResource(resource),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)

	// Set global trace provider
	otel.SetTracerProvider(provider)

	// Set global propagator for distributed tracing
	otel.SetTextMapPropagator(propagation.TraceContext{})

	// Get tracer instance
	tracer := provider.Tracer(serviceName)

	slog.Info("OtelTracer created successfully",
		"service", serviceName,
		"endpoint", endpoint,
		"tracer_type", fmt.Sprintf("%T", tracer))

	return &OtelTracer{
		tracer:   tracer,
		provider: provider,
		exporter: exporter,
	}, nil
}

func (ot *OtelTracer) Start(
	ctx context.Context, operationName string,
) (context.Context, tracing.Span, func()) {
	slog.Info("Starting span", "operation", operationName)
	newCtx, span := ot.tracer.Start(ctx, operationName)

	spanID := span.SpanContext().SpanID().String()
	traceID := span.SpanContext().TraceID().String()

	slog.Info("Span created",
		"operation", operationName,
		"span_id", spanID,
		"trace_id", traceID,
		"valid", span.SpanContext().IsValid())

	otelSpan := &OtelSpan{span: span}

	cancel := func() {
		slog.Info("Ending span",
			"operation", operationName,
			"span_id", spanID,
			"trace_id", traceID)
		span.End()

		// Force flush after each span for debugging
		if ot.provider != nil {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := ot.provider.ForceFlush(ctx); err != nil {
				slog.Error("Failed to flush spans", "error", err)
			} else {
				slog.Info("Successfully flushed spans", "span_id", spanID)
			}
		}
	}

	return newCtx, otelSpan, cancel
}

func (ot *OtelTracer) Close() error {
	slog.Info("Shutting down OtelTracer")

	if ot.provider != nil {
		slog.Info("Shutting down TracerProvider")
		if err := ot.provider.Shutdown(context.Background()); err != nil {
			slog.Error("Failed to shutdown trace provider", "error", err)
			return fmt.Errorf("failed to shutdown trace provider: %w", err)
		}
		slog.Info("TracerProvider shutdown completed")
	}

	if ot.exporter != nil {
		slog.Info("Shutting down exporter")
		if err := ot.exporter.Shutdown(context.Background()); err != nil {
			slog.Error("Failed to shutdown exporter", "error", err)
			return fmt.Errorf("failed to shutdown exporter: %w", err)
		}
		slog.Info("Exporter shutdown completed")
	}

	slog.Info("OtelTracer shutdown completed successfully")
	return nil
}

func (os *OtelSpan) SetError(err error, message string) {
	os.span.SetStatus(codes.Error, message)
	os.span.SetAttributes(
		attribute.String("error.message", message),
	)
	if err != nil {
		os.span.SetAttributes(
			attribute.String("error.object", err.Error()),
		)
	}
}

func (os *OtelSpan) SetTag(key string, value interface{}) {
	switch v := value.(type) {
	case string:
		os.span.SetAttributes(attribute.String(key, v))
	case int:
		os.span.SetAttributes(attribute.Int(key, v))
	case int64:
		os.span.SetAttributes(attribute.Int64(key, v))
	case bool:
		os.span.SetAttributes(attribute.Bool(key, v))
	case float64:
		os.span.SetAttributes(attribute.Float64(key, v))
	default:
		// Convert to string for other types
		os.span.SetAttributes(attribute.String(key, fmt.Sprintf("%v", v)))
	}
}

func (os *OtelSpan) GetSpanID() string {
	return os.span.SpanContext().SpanID().String()
}

func (os *OtelSpan) GetTraceID() string {
	return os.span.SpanContext().TraceID().String()
}
