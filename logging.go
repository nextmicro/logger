package logger

import (
	"context"
	"os"

	"github.com/mattn/go-colorable"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

type Logging struct {
	opt         Options
	atomicLevel zap.AtomicLevel
	lg          *zap.SugaredLogger
}

func New(opts ...Option) *Logging {
	opt := newOptions(opts...)
	l := &Logging{
		opt:         opt,
		atomicLevel: zap.NewAtomicLevelAt(opt.level.Level()),
	}
	if err := l.build(); err != nil {
		panic(err)
	}
	return l
}

func (l *Logging) build() error {
	var (
		sync []zapcore.WriteSyncer
	)

	switch l.opt.mode {
	case fileMode:
		file := l.buildFile()
		sync = append(sync, zapcore.AddSync(colorable.NewNonColorable(file)))
	case volumeMode:
		// TODO: 待实现
	default:
		sync = append(sync, zapcore.AddSync(os.Stdout))
	}

	var enc zapcore.Encoder
	if l.opt.encoder.IsConsole() {
		enc = zapcore.NewConsoleEncoder(l.opt.encoderConfig)
	} else {
		enc = zapcore.NewJSONEncoder(l.opt.encoderConfig)
	}

	zapLog := zap.New(zapcore.NewCore(enc, zapcore.NewMultiWriteSyncer(sync...), l.atomicLevel),
		zap.AddCaller(), zap.AddCallerSkip(l.opt.callerSkip)).Sugar()
	if len(l.opt.fields) > 0 {
		zapLog = zapLog.With(CopyFields(l.opt.fields)...)
	}
	if l.opt.namespace != "" {
		zapLog = zapLog.With(zap.Namespace(l.opt.namespace))
	}

	l.lg = zapLog

	return nil
}

func (l *Logging) buildFile() *lumberjack.Logger {
	return &lumberjack.Logger{
		Filename:   l.opt.filename,
		MaxSize:    l.opt.maxSize,
		MaxBackups: l.opt.maxBackups,
		MaxAge:     l.opt.maxAge,
		LocalTime:  l.opt.localTime,
		Compress:   l.opt.compress,
	}
}

func CopyFields(fields map[string]interface{}) []interface{} {
	dst := make([]interface{}, 0, len(fields)*2)
	for k, v := range fields {
		dst = append(dst, k, v)
	}
	return dst
}

func (l *Logging) WithContext(ctx context.Context) Logger {
	spanId := SpanID(ctx)
	traceId := TraceID(ctx)
	fields := make([]interface{}, 0, 4)
	if len(spanId) > 0 {
		fields = append(fields, spanKey, spanId)
	}
	if len(traceId) > 0 {
		fields = append(fields, traceKey, traceId)
	}

	logger := &Logging{
		opt:         l.opt,
		atomicLevel: l.atomicLevel,
		lg:          l.lg.With(fields...).WithOptions(zap.AddCallerSkip(0)),
	}
	return logger
}

func (l *Logging) WithFields(fields map[string]interface{}) Logger {
	return &Logging{
		opt:         l.opt,
		atomicLevel: l.atomicLevel,
		lg:          l.lg.With(CopyFields(fields)...).WithOptions(zap.AddCallerSkip(0)),
	}
}

func (l *Logging) WithCallDepth(callDepth int) Logger {
	return &Logging{
		opt:         l.opt,
		atomicLevel: l.atomicLevel,
		lg:          l.lg.WithOptions(zap.AddCallerSkip(callDepth)),
	}
}

func (l *Logging) Options() Options {
	return l.opt
}

func (l *Logging) SetLevel(lv Level) {
	l.atomicLevel.SetLevel(lv.Level())
}

func (l *Logging) Clone() *Logging {
	_copy := *l
	return &_copy
}

func (l *Logging) Debug(args ...interface{}) {
	l.lg.Debug(args...)
}

func (l *Logging) Info(args ...interface{}) {
	l.lg.Info(args...)
}

func (l *Logging) Warn(args ...interface{}) {
	l.lg.Warn(args...)
}

func (l *Logging) Error(args ...interface{}) {
	l.lg.Error(args...)
}

func (l *Logging) Fatal(args ...interface{}) {
	l.lg.Fatal(args...)
}

func (l *Logging) Debugf(template string, args ...interface{}) {
	l.lg.Debugf(template, args...)
}

func (l *Logging) Infof(template string, args ...interface{}) {
	l.lg.Infof(template, args...)
}

func (l *Logging) Warnf(template string, args ...interface{}) {
	l.lg.Warnf(template, args...)
}

func (l *Logging) Errorf(template string, args ...interface{}) {
	l.lg.Errorf(template, args...)
}

func (l *Logging) Fatalf(template string, args ...interface{}) {
	l.lg.Fatalf(template, args...)
}

func (l *Logging) Debugw(msg string, keysAndValues ...interface{}) {
	l.lg.Debugw(msg, keysAndValues...)
}

func (l *Logging) Infow(msg string, keysAndValues ...interface{}) {
	l.lg.Infow(msg, keysAndValues...)
}

func (l *Logging) Warnw(msg string, keysAndValues ...interface{}) {
	l.lg.Warnw(msg, keysAndValues...)
}

func (l *Logging) Errorw(msg string, keysAndValues ...interface{}) {
	l.lg.Errorw(msg, keysAndValues...)
}

func (l *Logging) Fatalw(msg string, keysAndValues ...interface{}) {
	l.lg.Fatalw(msg, keysAndValues...)
}

func (l *Logging) Sync() error {
	return l.lg.Sync()
}
