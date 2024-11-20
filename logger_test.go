package logger

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func TestNopLogger(t *testing.T) {
	logger := NewNop()

	t.Run("basic levels", func(t *testing.T) {
		logger.Debug("foo", zap.String("k", "v"))
		logger.Info("bar", zap.Int("x", 42))
		logger.Warn("baz", zap.Strings("ks", []string{"a", "b"}))
		logger.Error("qux", zap.Error(errors.New("great sadness")))
	})

	t.Run("DPanic", func(t *testing.T) {
		logger.With(zap.String("component", "whatever")).DPanic("stuff")
	})

	t.Run("Panic", func(t *testing.T) {
		assert.Panics(t, func() {
			logger.Panic("great sadness")
		}, "Nop logger should still cause panics.")
	})
}

func TestNewExample(t *testing.T) {
	logger := slog.New(
		slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			AddSource: true,
			Level:     slog.LevelInfo,
		}),
	)
	logger.Info("finished",
		slog.Int("status", http.StatusOK),
	)
}

type CustomHandler struct{}

func (h *CustomHandler) Handle(ctx context.Context, entry *zapcore.CheckedEntry, fields []zapcore.Field) ([]zapcore.Field, error) {
	v, ok := ctx.Value("tid").(string)
	if ok {
		fields = append(fields, zap.String("tid", v))
	}
	return fields, nil
}

func TestHandler(t *testing.T) {
	logger := NewExample(WithHandler(&CustomHandler{}))
	ctx := context.WithValue(context.Background(), "tid", "123456")
	logger.InfoContext(ctx, "hello world")
}
