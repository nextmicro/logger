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
