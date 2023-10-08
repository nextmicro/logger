package logger_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/nextmicro/logger"
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
	logging := logger.New()
	ctx, span := otel.Tracer("gokit").Start(context.TODO(), "HTTP Client Get /api/get")
	defer span.End()

	spanContext := trace.SpanContextFromContext(ctx)
	t.Logf("trace_id: %s", spanContext.TraceID().String())
	t.Logf("span_id: %s", spanContext.SpanID().String())

	logging.WithContext(ctx).Info("TestDefault_WithContext")
}

func TestLogging_WithFields(t *testing.T) {
	logging := logger.New()
	logging.WithFields(map[string]interface{}{
		"age":   22,
		"order": 100,
	}).Info("TestDefault_WithFields")
}

func TestLogging_Debug(t *testing.T) {
	logging := logger.New()
	logging.WithFields(map[string]interface{}{
		"age":   22,
		"order": 100,
	}).Debug("TestDefault_WithFields")

	logging.SetLevel(logger.DebugLevel)

	logging.WithFields(map[string]interface{}{
		"age":   22,
		"order": 100,
	}).Debug("TestDefault_WithFields")
}

func TestLogger(t *testing.T) {
	// Create a new logger for testing
	log := logger.New()

	// Test logging at different levels
	log.Debug("Debug message")
	log.Info("Info message")
	log.Warn("Warning message")
	log.Error("Error message")
	//log.Fatal("Fatal message")

	// Test formatting
	log.Debugf("Debug message with format: %s", "formatted")
	log.Infof("Info message with format: %s", "formatted")
	log.Warnf("Warning message with format: %s", "formatted")
	log.Errorf("Error message with format: %s", "formatted")
	//log.Fatalf("Fatal message with format: %s", "formatted")

	// Test log with fields
	fields := map[string]interface{}{
		"user":   "john_doe",
		"status": "failed",
	}
	log.WithFields(fields).Info("User login failed")

	// Test setting log level
	log.SetLevel(logger.DebugLevel)
	log.Debug("Debug message should be visible")
	log.SetLevel(logger.InfoLevel)
	log.Debug("Debug message should not be visible")

	// Test syncing
	err := logger.Sync()
	if err != nil {
		t.Errorf("Error syncing logger: %v", err)
	}

	// Add more test cases as needed
}

func TestLoggerWithContext(t *testing.T) {
	// Create a context with some values for testing
	ctx := context.WithValue(context.Background(), "spanId", "123")
	ctx = context.WithValue(ctx, "traceId", "456")

	// Create a logger with context
	log := logger.DefaultLogger.WithContext(ctx)

	// Test logging with context-based fields
	log.Info("Logging with context")

	// Add more test cases as needed
}

func TestLoggerWithFields(t *testing.T) {
	// Create a logger with some initial fields
	fields := map[string]interface{}{
		"app":     "my_app",
		"version": "1.0",
	}
	logger := logger.DefaultLogger.WithFields(fields)

	// Test logging with additional fields
	additionalFields := map[string]interface{}{
		"user": "alice",
	}
	logger.WithFields(additionalFields).Info("Additional fields")

	// Add more test cases as needed
}

func TestFilename(t *testing.T) {
	logger.DefaultLogger = logger.New(
		logger.WithMode("file"),
		logger.WithMaxSize(0),
		logger.WithMaxBackups(1),
		logger.WithMaxAge(3),
		logger.WithLocalTime(true),
		logger.WithCompress(false),
		logger.WithFilename("./logs/test"),
	)

	defer logger.Sync()

	for i := 0; i < 1000; i++ {
		logger.Info("test msg")
	}
}

type CustomOutput struct {
}

func (c *CustomOutput) Write(p []byte) (n int, err error) {
	fmt.Println(string(p))
	return 0, nil
}

// 自定义输出源
func TestCustomOutput(t *testing.T) {
	logger.DefaultLogger = logger.New(
		logger.WithMode("custom"),
		logger.WithWriter(&CustomOutput{}),
	)

	defer logger.Sync()

	for i := 0; i < 1000; i++ {
		logger.Info("test msg")
	}
}
