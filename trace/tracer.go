package trace

import (
	"context"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	otel_trace "go.opentelemetry.io/otel/trace"
)

var (
	tracer        = otel.Tracer("Rizotto")
	NoSampledMask = ^otel_trace.FlagsSampled
)

func DisableTracing(ctx context.Context) (context.Context, otel_trace.Span) {
	ctx, span := Span(ctx, "disable-tracing")
	spanContext := otel_trace.SpanContextFromContext(ctx)
	curFlags := spanContext.TraceFlags()
	go func() {
		time.Sleep(time.Second)
		span.End()
	}()

	return otel_trace.ContextWithSpanContext(ctx, spanContext.WithTraceFlags(
		curFlags&NoSampledMask,
	)), span
}

//nolint:spancheck // caller owns the returned span and is responsible for calling span.End()
func Span(ctx context.Context, name string, opts ...otel_trace.SpanStartOption) (context.Context, otel_trace.Span) {
	ctx, span := tracer.Start(ctx, name, opts...)

	return ctx, span
}

func ServerKind() otel_trace.SpanStartOption {
	return otel_trace.WithSpanKind(otel_trace.SpanKindServer)
}

func ClientKind() otel_trace.SpanStartOption {
	return otel_trace.WithSpanKind(otel_trace.SpanKindClient)
}

var SpanFromContext = otel_trace.SpanFromContext

type KeyValue = attribute.KeyValue

var (
	String = attribute.String
	Int    = attribute.Int
	Bool   = attribute.Bool
)
