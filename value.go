package logger

import (
	"context"

	"go.opentelemetry.io/otel/trace"
)

// TraceID returns a traceid valuer.
func TraceID(ctx context.Context) string {
	if span := trace.SpanContextFromContext(ctx); span.HasTraceID() {
		return span.TraceID().String()
	}
	return ""
}

// SpanID returns a spanid valuer.
func SpanID(ctx context.Context) string {
	if span := trace.SpanContextFromContext(ctx); span.HasSpanID() {
		return span.SpanID().String()
	}
	return ""
}
