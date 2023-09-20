package logger

import (
	"context"
	"testing"

	"github.com/go-volo/logger"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

func TestMain(t *testing.M) {
	exporter, err := jaeger.New(jaeger.WithCollectorEndpoint())
	if err != nil {
		panic(err)
	}

	options := make([]sdktrace.TracerProviderOption, 0)
	options = append(options, sdktrace.WithBatcher(exporter))
	provider := sdktrace.NewTracerProvider(options...)
	otel.SetTracerProvider(provider)

	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{}, propagation.Baggage{}))
	otel.SetErrorHandler(otel.ErrorHandlerFunc(func(err error) {
		logger.Errorf("[otel] error: %v", err)
	}))

	t.Run()
}

func TestLogging_WithContext(t *testing.T) {
	logging := New()
	ctx, span := otel.Tracer("gokit").Start(context.TODO(), "HTTP Client Get /api/get")
	defer span.End()

	spanContext := trace.SpanContextFromContext(ctx)
	t.Logf("trace_id: %s", spanContext.TraceID().String())
	t.Logf("span_id: %s", spanContext.SpanID().String())

	logging.WithContext(ctx).Info("TestDefault_WithContext")
}

func TestLogging_WithFields(t *testing.T) {
	logging := New()
	logging.WithFields(map[string]interface{}{
		"age":   22,
		"order": 100,
	}).Info("TestDefault_WithFields")
}

func TestLogging_Debug(t *testing.T) {
	logging := New()
	logging.WithFields(map[string]interface{}{
		"age":   22,
		"order": 100,
	}).Debug("TestDefault_WithFields")

	logging.SetLevel(DebugLevel)

	logging.WithFields(map[string]interface{}{
		"age":   22,
		"order": 100,
	}).Debug("TestDefault_WithFields")
}
