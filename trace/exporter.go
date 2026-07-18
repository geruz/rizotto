package trace

import (
	"context"
	"os"

	"github.com/geruz/rizotto/logger"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"

	trace_sdk "go.opentelemetry.io/otel/sdk/trace"
)

func MustInit(
	ctx context.Context,
	attributes map[string]string,
) {
	exp, err := otlptracehttp.New(ctx)
	if err != nil {
		logger.Error(ctx, "failed to initialize jaeger exporter", err)

		return
	}
	host, _ := os.Hostname()
	attr := make([]attribute.KeyValue, 0, len(attributes)+1)
	attr = append(attr, attribute.String("host", host))
	for k, v := range attributes {
		attr = append(attr, attribute.String(k, v))
	}
	otel.SetTracerProvider(trace_sdk.NewTracerProvider(
		trace_sdk.WithBatcher(exp),
		trace_sdk.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			attr...,
		)),
	))
}
