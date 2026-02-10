// Copyright (c) 2016 Uber Technologies, Inc.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package logger

import (
	"context"
	"fmt"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"go.uber.org/multierr"
)

const (
	_oddNumberErrMsg    = "Ignored key without a value."
	_nonStringKeyErrMsg = "Ignored key-value pairs with non-string keys."
	_multipleErrMsg     = "Multiple errors without a key."
)

// A SugaredLogger wraps the base Logger functionality in a slower, but less
// verbose, API. Any Logger can be converted to a SugaredLogger with its Sugar
// method.
//
// Unlike the Logger, the SugaredLogger doesn't insist on structured logging.
// For each log level, it exposes four methods:
//
//   - methods named after the log level for log.Print-style logging
//   - methods ending in "w" for loosely-typed structured logging
//   - methods ending in "f" for log.Printf-style logging
//   - methods ending in "ln" for log.Println-style logging
//
// For example, the methods for InfoLevel are:
//
//	Info(...any)           Print-style logging
//	Infow(...any)          Structured logging (read as "info with")
//	Infof(string, ...any)  Printf-style logging
//	Infoln(...any)         Println-style logging
type SugaredLogger struct {
	base *Logger
}

// Desugar unwraps a SugaredLogger, exposing the original Logger. Desugaring
// is quite inexpensive, so it's reasonable for a single application to use
// both Loggers and SugaredLoggers, converting between them on the boundaries
// of performance-sensitive code.
func (s *SugaredLogger) Desugar() *Logger {
	base := s.base.clone()
	base.callerSkip -= 2
	return base
}

// Named adds a sub-scope to the logger's name. See Logger.Named for details.
func (s *SugaredLogger) Named(name string) *SugaredLogger {
	return &SugaredLogger{base: s.base.Named(name)}
}

// WithOptions clones the current SugaredLogger, applies the supplied Options,
// and returns the result. It's safe to use concurrently.
func (s *SugaredLogger) WithOptions(opts ...Option) *SugaredLogger {
	base := s.base.clone()
	for _, opt := range opts {
		opt.apply(base)
	}
	return &SugaredLogger{base: base}
}

// With adds a variadic number of fields to the logging context. It accepts a
// mix of strongly-typed Field objects and loosely-typed key-value pairs. When
// processing pairs, the first element of the pair is used as the field key
// and the second as the field value.
//
// For example,
//
//	 sugaredLogger.With(
//	   "hello", "world",
//	   "failure", errors.New("oh no"),
//	   Stack(),
//	   "count", 42,
//	   "user", User{Name: "alice"},
//	)
//
// is the equivalent of
//
//	unsugared.With(
//	  String("hello", "world"),
//	  String("failure", "oh no"),
//	  Stack(),
//	  Int("count", 42),
//	  Object("user", User{Name: "alice"}),
//	)
//
// Note that the keys in key-value pairs should be strings. In development,
// passing a non-string key panics. In production, the logger is more
// forgiving: a separate error is logged, but the key-value pair is skipped
// and execution continues. Passing an orphaned key triggers similar behavior:
// panics in development and errors in production.
func (s *SugaredLogger) With(args ...interface{}) *SugaredLogger {
	return &SugaredLogger{base: s.base.With(s.sweetenFields(args)...)}
}

// WithLazy adds a variadic number of fields to the logging context lazily.
// The fields are evaluated only if the logger is further chained with [With]
// or is written to with any of the log level methods.
// Until that occurs, the logger may retain references to objects inside the fields,
// and logging will reflect the state of an object at the time of logging,
// not the time of WithLazy().
//
// Similar to [With], fields added to the child don't affect the parent,
// and vice versa. Also, the keys in key-value pairs should be strings. In development,
// passing a non-string key panics, while in production it logs an error and skips the pair.
// Passing an orphaned key has the same behavior.
func (s *SugaredLogger) WithLazy(args ...interface{}) *SugaredLogger {
	return &SugaredLogger{base: s.base.WithLazy(s.sweetenFields(args)...)}
}

// Level reports the minimum enabled level for this logger.
//
// For NopLoggers, this is [zapcore.InvalidLevel].
func (s *SugaredLogger) Level() zapcore.Level {
	return zapcore.LevelOf(s.base.core)
}

// Log logs the provided arguments at provided level.
// Spaces are added between arguments when neither is a string.
func (s *SugaredLogger) Log(lvl zapcore.Level, args ...interface{}) {
	s.log(context.Background(), lvl, "", args, nil)
}

// LogContext logs the provided arguments at provided level.
func (s *SugaredLogger) LogContext(ctx context.Context, lvl zapcore.Level, args ...interface{}) {
	s.log(ctx, lvl, "", args, nil)
}

// Debug logs the provided arguments at [DebugLevel].
// Spaces are added between arguments when neither is a string.
func (s *SugaredLogger) Debug(args ...interface{}) {
	s.log(context.Background(), zapcore.DebugLevel, "", args, nil)
}

// DebugContext logs the provided arguments at [DebugLevel].
func (s *SugaredLogger) DebugContext(ctx context.Context, args ...interface{}) {
	s.log(ctx, zapcore.DebugLevel, "", args, nil)
}

// Info logs the provided arguments at [InfoLevel].
// Spaces are added between arguments when neither is a string.
func (s *SugaredLogger) Info(args ...interface{}) {
	s.log(context.Background(), zapcore.InfoLevel, "", args, nil)
}

// InfoContext logs the provided arguments at [InfoLevel].
func (s *SugaredLogger) InfoContext(ctx context.Context, args ...interface{}) {
	s.log(ctx, zapcore.InfoLevel, "", args, nil)
}

// Warn logs the provided arguments at [WarnLevel].
// Spaces are added between arguments when neither is a string.
func (s *SugaredLogger) Warn(args ...interface{}) {
	s.log(context.Background(), zapcore.WarnLevel, "", args, nil)
}

// WarnContext logs the provided arguments at [WarnLevel].
// Spaces are added between arguments when neither is a string.
func (s *SugaredLogger) WarnContext(ctx context.Context, args ...interface{}) {
	s.log(ctx, zapcore.WarnLevel, "", args, nil)
}

// Error logs the provided arguments at [ErrorLevel].
// Spaces are added between arguments when neither is a string.
func (s *SugaredLogger) Error(args ...interface{}) {
	s.log(context.Background(), zapcore.ErrorLevel, "", args, nil)
}

// ErrorContext logs the provided arguments at [ErrorLevel].
// Spaces are added between arguments when neither is a string.
func (s *SugaredLogger) ErrorContext(ctx context.Context, args ...interface{}) {
	s.log(ctx, zapcore.ErrorLevel, "", args, nil)
}

// DPanic logs the provided arguments at [DPanicLevel].
// In development, the logger then panics. (See [DPanicLevel] for details.)
// Spaces are added between arguments when neither is a string.
func (s *SugaredLogger) DPanic(args ...interface{}) {
	s.log(context.Background(), zapcore.DPanicLevel, "", args, nil)
}

// DPanicContext logs the provided arguments at [DPanicLevel].
// Spaces are added between arguments when neither is a string.
func (s *SugaredLogger) DPanicContext(ctx context.Context, args ...interface{}) {
	s.log(ctx, zapcore.DPanicLevel, "", args, nil)
}

// Panic constructs a message with the provided arguments and panics.
// Spaces are added between arguments when neither is a string.
func (s *SugaredLogger) Panic(args ...interface{}) {
	s.log(context.Background(), zapcore.PanicLevel, "", args, nil)
}

// PanicContext constructs a message with the provided arguments and panics.
// Spaces are added between arguments when neither is a string.
func (s *SugaredLogger) PanicContext(ctx context.Context, args ...interface{}) {
	s.log(ctx, zapcore.PanicLevel, "", args, nil)
}

// Fatal constructs a message with the provided arguments and calls os.Exit.
// Spaces are added between arguments when neither is a string.
func (s *SugaredLogger) Fatal(args ...interface{}) {
	s.log(context.Background(), zapcore.FatalLevel, "", args, nil)
}

// FatalContext constructs a message with the provided arguments and calls os.Exit.
// Spaces are added between arguments when neither is a string.
func (s *SugaredLogger) FatalContext(ctx context.Context, args ...interface{}) {
	s.log(ctx, zapcore.FatalLevel, "", args, nil)
}

// Logf formats the message according to the format specifier
// and logs it at provided level.
func (s *SugaredLogger) Logf(lvl zapcore.Level, template string, args ...interface{}) {
	s.log(context.Background(), lvl, template, args, nil)
}

// LogContextf formats the message according to the format specifier
// and logs it at provided level.
func (s *SugaredLogger) LogContextf(ctx context.Context, lvl zapcore.Level, template string, args ...interface{}) {
	s.log(ctx, lvl, template, args, nil)
}

// Debugf formats the message according to the format specifier
// and logs it at [DebugLevel].
func (s *SugaredLogger) Debugf(template string, args ...interface{}) {
	s.log(context.Background(), zapcore.DebugLevel, template, args, nil)
}

// DebugContextf formats the message according to the format specifier
// and logs it at [DebugLevel].
func (s *SugaredLogger) DebugContextf(ctx context.Context, template string, args ...interface{}) {
	s.log(ctx, zapcore.DebugLevel, template, args, nil)
}

// Infof formats the message according to the format specifier
// and logs it at [InfoLevel].
func (s *SugaredLogger) Infof(template string, args ...interface{}) {
	s.log(context.Background(), zapcore.InfoLevel, template, args, nil)
}

// InfoContextf formats the message according to the format specifier
// and logs it at [InfoLevel].
func (s *SugaredLogger) InfoContextf(ctx context.Context, template string, args ...interface{}) {
	s.log(ctx, zapcore.InfoLevel, template, args, nil)
}

// Warnf formats the message according to the format specifier
// and logs it at [WarnLevel].
func (s *SugaredLogger) Warnf(template string, args ...interface{}) {
	s.log(context.Background(), zapcore.WarnLevel, template, args, nil)
}

// WarnContextf formats the message according to the format specifier
// and logs it at [WarnLevel].
func (s *SugaredLogger) WarnContextf(ctx context.Context, template string, args ...interface{}) {
	s.log(ctx, zapcore.WarnLevel, template, args, nil)
}

// Errorf formats the message according to the format specifier
// and logs it at [ErrorLevel].
func (s *SugaredLogger) Errorf(template string, args ...interface{}) {
	s.log(context.Background(), zapcore.ErrorLevel, template, args, nil)
}

// ErrorContextf formats the message according to the format specifier
// and logs it at [ErrorLevel].
func (s *SugaredLogger) ErrorContextf(ctx context.Context, template string, args ...interface{}) {
	s.log(ctx, zapcore.ErrorLevel, template, args, nil)
}

// DPanicf formats the message according to the format specifier
// and logs it at [DPanicLevel].
// In development, the logger then panics. (See [DPanicLevel] for details.)
func (s *SugaredLogger) DPanicf(template string, args ...interface{}) {
	s.log(context.Background(), zapcore.DPanicLevel, template, args, nil)
}

// DPanicContextf formats the message according to the format specifier
// and logs it at [DPanicLevel].
// In development, the logger then panics. (See [DPanicLevel] for details.)
func (s *SugaredLogger) DPanicContextf(ctx context.Context, template string, args ...interface{}) {
	s.log(ctx, zapcore.DPanicLevel, template, args, nil)
}

// Panicf formats the message according to the format specifier
// and panics.
func (s *SugaredLogger) Panicf(template string, args ...interface{}) {
	s.log(context.Background(), zapcore.PanicLevel, template, args, nil)
}

// PanicContextf formats the message according to the format specifier
// and panics.
func (s *SugaredLogger) PanicContextf(ctx context.Context, template string, args ...interface{}) {
	s.log(ctx, zapcore.PanicLevel, template, args, nil)
}

// Fatalf formats the message according to the format specifier
// and calls os.Exit.
func (s *SugaredLogger) Fatalf(template string, args ...interface{}) {
	s.log(context.Background(), zapcore.FatalLevel, template, args, nil)
}

// FatalfContext formats the message according to the format specifier
// and calls os.Exit.
func (s *SugaredLogger) FatalfContext(ctx context.Context, template string, args ...interface{}) {
	s.log(ctx, zapcore.FatalLevel, template, args, nil)
}

// Logw logs a message with some additional context. The variadic key-value
// pairs are treated as they are in With.
func (s *SugaredLogger) Logw(lvl zapcore.Level, msg string, keysAndValues ...interface{}) {
	s.log(context.Background(), lvl, msg, nil, keysAndValues)
}

// LogwContext logs a message with some additional context. The variadic key-value
// pairs are treated as they are in With.
func (s *SugaredLogger) LogwContext(ctx context.Context, lvl zapcore.Level, msg string, keysAndValues ...interface{}) {
	s.log(ctx, lvl, msg, nil, keysAndValues)
}

// Debugw logs a message with some additional context. The variadic key-value
// pairs are treated as they are in With.
//
// When debug-level logging is disabled, this is much faster than
//
//	s.With(keysAndValues).Debug(msg)
func (s *SugaredLogger) Debugw(msg string, keysAndValues ...interface{}) {
	s.log(context.Background(), zapcore.DebugLevel, msg, nil, keysAndValues)
}

// DebugwContext logs a message with some additional context. The variadic key-value
// pairs are treated as they are in With.
//
// When debug-level logging is disabled, this is much faster than
//
//	s.With(keysAndValues).Debug(msg)
func (s *SugaredLogger) DebugwContext(ctx context.Context, msg string, keysAndValues ...interface{}) {
	s.log(ctx, zapcore.DebugLevel, msg, nil, keysAndValues)
}

// Infow logs a message with some additional context. The variadic key-value
// pairs are treated as they are in With.
func (s *SugaredLogger) Infow(msg string, keysAndValues ...interface{}) {
	s.log(context.Background(), zapcore.InfoLevel, msg, nil, keysAndValues)
}

// InfowContext logs a message with some additional context. The variadic key-value
// pairs are treated as they are in With.
func (s *SugaredLogger) InfowContext(ctx context.Context, msg string, keysAndValues ...interface{}) {
	s.log(ctx, zapcore.InfoLevel, msg, nil, keysAndValues)
}

// Warnw logs a message with some additional context. The variadic key-value
// pairs are treated as they are in With.
func (s *SugaredLogger) Warnw(msg string, keysAndValues ...interface{}) {
	s.log(context.Background(), zapcore.WarnLevel, msg, nil, keysAndValues)
}

// WarnwContext logs a message with some additional context. The variadic key-value
// pairs are treated as they are in With.
func (s *SugaredLogger) WarnwContext(ctx context.Context, msg string, keysAndValues ...interface{}) {
	s.log(ctx, zapcore.WarnLevel, msg, nil, keysAndValues)
}

// Errorw logs a message with some additional context. The variadic key-value
// pairs are treated as they are in With.
func (s *SugaredLogger) Errorw(msg string, keysAndValues ...interface{}) {
	s.log(context.Background(), zapcore.ErrorLevel, msg, nil, keysAndValues)
}

// ErrorwContext logs a message with some additional context. The variadic key-value
// pairs are treated as they are in With.
func (s *SugaredLogger) ErrorwContext(ctx context.Context, msg string, keysAndValues ...interface{}) {
	s.log(ctx, zapcore.ErrorLevel, msg, nil, keysAndValues)
}

// DPanicw logs a message with some additional context. In development, the
// logger then panics. (See DPanicLevel for details.) The variadic key-value
// pairs are treated as they are in With.
func (s *SugaredLogger) DPanicw(msg string, keysAndValues ...interface{}) {
	s.log(context.Background(), zapcore.DPanicLevel, msg, nil, keysAndValues)
}

// DPanicwContext logs a message with some additional context. In development, the
// logger then panics. (See DPanicLevel for details.) The variadic key-value
// pairs are treated as they are in With.
func (s *SugaredLogger) DPanicwContext(ctx context.Context, msg string, keysAndValues ...interface{}) {
	s.log(ctx, zapcore.DPanicLevel, msg, nil, keysAndValues)
}

// Panicw logs a message with some additional context, then panics. The
// variadic key-value pairs are treated as they are in With.
func (s *SugaredLogger) Panicw(msg string, keysAndValues ...interface{}) {
	s.log(context.Background(), zapcore.PanicLevel, msg, nil, keysAndValues)
}

// PanicwContext logs a message with some additional context, then panics. The
// variadic key-value pairs are treated as they are in With.
func (s *SugaredLogger) PanicwContext(ctx context.Context, msg string, keysAndValues ...interface{}) {
	s.log(ctx, zapcore.PanicLevel, msg, nil, keysAndValues)
}

// Fatalw logs a message with some additional context, then calls os.Exit. The
// variadic key-value pairs are treated as they are in With.
func (s *SugaredLogger) Fatalw(msg string, keysAndValues ...interface{}) {
	s.log(context.Background(), zapcore.FatalLevel, msg, nil, keysAndValues)
}

// FatalwContext logs a message with some additional context, then calls os.Exit. The
// variadic key-value pairs are treated as they are in With.
func (s *SugaredLogger) FatalwContext(ctx context.Context, msg string, keysAndValues ...interface{}) {
	s.log(ctx, zapcore.FatalLevel, msg, nil, keysAndValues)
}

// Logln logs a message at provided level.
// Spaces are always added between arguments.
func (s *SugaredLogger) Logln(lvl zapcore.Level, args ...interface{}) {
	s.logln(context.Background(), lvl, args, nil)
}

func (s *SugaredLogger) LoglnContext(ctx context.Context, lvl zapcore.Level, args ...interface{}) {
	s.logln(ctx, lvl, args, nil)
}

// Debugln logs a message at [DebugLevel].
// Spaces are always added between arguments.
func (s *SugaredLogger) Debugln(args ...interface{}) {
	s.logln(context.Background(), zapcore.DebugLevel, args, nil)
}

func (s *SugaredLogger) DebuglnContext(ctx context.Context, args ...interface{}) {
	s.logln(ctx, zapcore.DebugLevel, args, nil)
}

// Infoln logs a message at [InfoLevel].
// Spaces are always added between arguments.
func (s *SugaredLogger) Infoln(args ...interface{}) {
	s.logln(context.Background(), zapcore.InfoLevel, args, nil)
}

func (s *SugaredLogger) InfolnContext(ctx context.Context, args ...interface{}) {
	s.logln(ctx, zapcore.InfoLevel, args, nil)
}

// Warnln logs a message at [WarnLevel].
// Spaces are always added between arguments.
func (s *SugaredLogger) Warnln(args ...interface{}) {
	s.logln(context.Background(), zapcore.WarnLevel, args, nil)
}

func (s *SugaredLogger) WarnlnContext(ctx context.Context, args ...interface{}) {
	s.logln(ctx, zapcore.WarnLevel, args, nil)
}

// Errorln logs a message at [ErrorLevel].
// Spaces are always added between arguments.
func (s *SugaredLogger) Errorln(args ...interface{}) {
	s.logln(context.Background(), zapcore.ErrorLevel, args, nil)
}

func (s *SugaredLogger) ErrorlnContext(ctx context.Context, args ...interface{}) {
	s.logln(ctx, zapcore.ErrorLevel, args, nil)
}

// DPanicln logs a message at [DPanicLevel].
// In development, the logger then panics. (See [DPanicLevel] for details.)
// Spaces are always added between arguments.
func (s *SugaredLogger) DPanicln(args ...interface{}) {
	s.logln(context.Background(), zapcore.DPanicLevel, args, nil)
}

func (s *SugaredLogger) DPaniclnContext(ctx context.Context, args ...interface{}) {
	s.logln(ctx, zapcore.DPanicLevel, args, nil)
}

// Panicln logs a message at [PanicLevel] and panics.
// Spaces are always added between arguments.
func (s *SugaredLogger) Panicln(args ...interface{}) {
	s.logln(context.Background(), zapcore.PanicLevel, args, nil)
}

func (s *SugaredLogger) PaniclnContext(ctx context.Context, args ...interface{}) {
	s.logln(ctx, zapcore.PanicLevel, args, nil)
}

// Fatalln logs a message at [FatalLevel] and calls os.Exit.
// Spaces are always added between arguments.
func (s *SugaredLogger) Fatalln(args ...interface{}) {
	s.logln(context.Background(), zapcore.FatalLevel, args, nil)
}

func (s *SugaredLogger) FatallnContext(ctx context.Context, args ...interface{}) {
	s.logln(ctx, zapcore.FatalLevel, args, nil)
}

// Sync flushes any buffered log entries.
func (s *SugaredLogger) Sync() error {
	return s.base.Sync()
}

// log message with Sprint, Sprintf, or neither.
func (s *SugaredLogger) log(ctx context.Context, lvl zapcore.Level, template string, fmtArgs []interface{}, context []interface{}) {
	// If logging at this level is completely disabled, skip the overhead of
	// string formatting.
	if lvl < zapcore.DPanicLevel && !s.base.Core().Enabled(lvl) {
		return
	}

	msg := getMessage(template, fmtArgs)
	if ce := s.base.Check(lvl, msg); ce != nil {
		fields := s.sweetenFields(context)
		_fields, err := s.base.handler.Handle(ctx, ce, fields)
		if err != nil {
			s.Error("Error occurred while handling log context", zap.Error(err))
		} else {
			fields = append(fields, _fields...)
		}
		ce.Write(fields...)
	}
}

// logln message with Sprintln
func (s *SugaredLogger) logln(ctx context.Context, lvl zapcore.Level, fmtArgs []interface{}, context []interface{}) {
	if lvl < zapcore.DPanicLevel && !s.base.Core().Enabled(lvl) {
		return
	}

	msg := getMessageln(fmtArgs)
	if ce := s.base.Check(lvl, msg); ce != nil {
		fields := s.sweetenFields(context)
		_fields, err := s.base.handler.Handle(ctx, ce, fields)
		if err != nil {
			s.Error("Error occurred while handling log context", zap.Error(err))
		} else {
			fields = append(fields, _fields...)
		}
		ce.Write(fields...)
	}
}

// getMessage format with Sprint, Sprintf, or neither.
func getMessage(template string, fmtArgs []interface{}) string {
	if len(fmtArgs) == 0 {
		return template
	}

	if template != "" {
		return fmt.Sprintf(template, fmtArgs...)
	}

	if len(fmtArgs) == 1 {
		if str, ok := fmtArgs[0].(string); ok {
			return str
		}
	}
	return fmt.Sprint(fmtArgs...)
}

// getMessageln format with Sprintln.
func getMessageln(fmtArgs []interface{}) string {
	msg := fmt.Sprintln(fmtArgs...)
	return msg[:len(msg)-1]
}

func (s *SugaredLogger) sweetenFields(args []interface{}) []zapcore.Field {
	if len(args) == 0 {
		return nil
	}

	var (
		// Allocate enough space for the worst case; if users pass only structured
		// fields, we shouldn't penalize them with extra allocations.
		fields    = make([]zapcore.Field, 0, len(args))
		invalid   invalidPairs
		seenError bool
	)

	for i := 0; i < len(args); {
		// This is a strongly-typed field. Consume it and move on.
		if f, ok := args[i].(zapcore.Field); ok {
			fields = append(fields, f)
			i++
			continue
		}

		// If it is an error, consume it and move on.
		if err, ok := args[i].(error); ok {
			if !seenError {
				seenError = true
				fields = append(fields, zap.Error(err))
			} else {
				s.base.Error(_multipleErrMsg, zap.Error(err))
			}
			i++
			continue
		}

		// Make sure this element isn't a dangling key.
		if i == len(args)-1 {
			s.base.Error(_oddNumberErrMsg, zap.Any("ignored", args[i]))
			break
		}

		// Consume this value and the next, treating them as a key-value pair. If the
		// key isn't a string, add this pair to the slice of invalid pairs.
		key, val := args[i], args[i+1]
		if keyStr, ok := key.(string); !ok {
			// Subsequent errors are likely, so allocate once up front.
			if cap(invalid) == 0 {
				invalid = make(invalidPairs, 0, len(args)/2)
			}
			invalid = append(invalid, invalidPair{i, key, val})
		} else {
			fields = append(fields, zap.Any(keyStr, val))
		}
		i += 2
	}

	// If we encountered any invalid key-value pairs, log an error.
	if len(invalid) > 0 {
		s.base.Error(_nonStringKeyErrMsg, zap.Array("invalid", invalid))
	}
	return fields
}

type invalidPair struct {
	position   int
	key, value interface{}
}

func (p invalidPair) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddInt64("position", int64(p.position))
	zap.Any("key", p.key).AddTo(enc)
	zap.Any("value", p.value).AddTo(enc)
	return nil
}

type invalidPairs []invalidPair

func (ps invalidPairs) MarshalLogArray(enc zapcore.ArrayEncoder) error {
	var err error
	for i := range ps {
		err = multierr.Append(err, enc.AppendObject(ps[i]))
	}
	return err
}
