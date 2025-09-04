package tracer

import (
	"context"
	"time"

	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/support/log"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
)

type StellarTracer struct {
	OtelEndpoint   string
	ServiceName    string
	ServiceVersion string
}

// NewStellarTracer returns updated stellar tracer object with service and endpoint details
func NewStellarTracer(OtelEndpoint, ServiceName, ServiceVersion string) *StellarTracer {
	return &StellarTracer{
		OtelEndpoint:   OtelEndpoint,
		ServiceName:    ServiceName,
		ServiceVersion: ServiceVersion,
	}
}

// InitializeTracer sets up traceProvider and returns a function to handle traceprovider
func (stellarTracer *StellarTracer) InitializeTracer() (func(), error) {
	log.Infof("Initializing tracer")
	headers := map[string]string{
		"content-type": "application/json",
	}

	exporter, err := otlptrace.New(
		context.Background(),
		otlptracehttp.NewClient(
			otlptracehttp.WithEndpoint(stellarTracer.OtelEndpoint),
			otlptracehttp.WithHeaders(headers),
			otlptracehttp.WithInsecure(),
		),
	)
	if err != nil {
		return nil, errors.Wrap(err, "Error while creating exporter")
	}

	res, err := resource.New(
		context.Background(),
		resource.WithAttributes(
			semconv.ServiceName(stellarTracer.ServiceName),
			semconv.ServiceVersion(stellarTracer.ServiceVersion),
		),
	)

	if err != nil {
		return nil, errors.Wrap(err, "failed to create exporter")
	}

	traceProvider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)

	// Set traceprovider for the otel.
	otel.SetTracerProvider(traceProvider)
	otel.SetTextMapPropagator(propagation.TraceContext{})

	return func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := traceProvider.Shutdown(ctx); err != nil {
			log.Error("Error shutting down tracer provider", err)
		}
	}, nil
}
