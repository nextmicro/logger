package logger

import (
	"context"

	"go.uber.org/zap/zapcore"
)

type Handler interface {
	// Handle serializes the Entry and any Fields supplied at the log site and
	// writes them to their destination.
	//
	// If called, Write should always log the Entry and Fields; it should not
	// replicate the logic of Check.
	Handle(context.Context, *zapcore.CheckedEntry, []zapcore.Field) ([]zapcore.Field, error)
}

// CommonHandler wraps a Handler and implements Handler.
type CommonHandler struct {
	handler Handler
}

// NewCommonHandler returns a new CommonHandler wrapping h.
func NewCommonHandler(h Handler) Handler {
	return &CommonHandler{handler: h}
}

// Handler returns l's Handler.
func (h *CommonHandler) Handler() Handler { return h.handler }

func (h *CommonHandler) Handle(ctx context.Context, entry *zapcore.CheckedEntry, fields []zapcore.Field) ([]zapcore.Field, error) {
	if h.Handler() == nil {
		return fields, nil
	}
	return h.Handler().Handle(ctx, entry, fields)
}
