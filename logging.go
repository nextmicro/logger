package logger

import (
	"context"
	"io"
	"os"
	"path"

	"github.com/mattn/go-colorable"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var _ Logger = (*Logging)(nil)

// DefaultLogger is default logger.
var DefaultLogger Logger = New()

type Logging struct {
	opt         Options
	atomicLevel zap.AtomicLevel
	lg          *zap.SugaredLogger

	_rollingFiles []io.Writer
}

// WrappedWriteSyncer is a helper struct implementing zapcore.WriteSyncer to
// wrap a standard os.Stdout handle, giving control over the WriteSyncer's
// Sync() function. Sync() results in an error on Windows in combination with
// os.Stdout ("sync /dev/stdout: The handle is invalid."). WrappedWriteSyncer
// simply does nothing when Sync() is called by Zap.
type WrappedWriteSyncer struct {
	file io.Writer
}

func (mws WrappedWriteSyncer) Write(p []byte) (n int, err error) {
	return mws.file.Write(p)
}
func (mws WrappedWriteSyncer) Sync() error {
	return nil
}

func New(opts ...Option) *Logging {
	opt := newOptions(opts...)
	l := &Logging{
		opt:         opt,
		atomicLevel: zap.NewAtomicLevelAt(opt.level.unmarshalZapLevel()),
	}
	if err := l.build(); err != nil {
		panic(err)
	}
	return l
}

func (l *Logging) LevelEnablerFunc(level zapcore.Level) LevelEnablerFunc {
	return func(lvl zapcore.Level) bool {
		if level == zapcore.FatalLevel {
			return l.atomicLevel.Level() <= level && lvl >= level
		}
		return l.atomicLevel.Level() <= level && lvl == level
	}
}

func (l *Logging) build() error {
	var (
		cores []zapcore.Core
	)

	switch l.opt.mode {
	case FileMode:
		var _cores []zapcore.Core
		if l.opt.writer != nil {
			_cores = l.buildCustomWriter()
		} else if l.opt.filename != "" {
			_cores = l.buildFile()
		} else {
			_cores = l.buildFiles()
		}
		if len(_cores) > 0 {
			cores = append(cores, _cores...)
		}
	default:
		_cores := l.buildConsole()
		if len(_cores) > 0 {
			cores = append(cores, _cores...)
		}
	}

	zapLog := zap.New(zapcore.NewTee(cores...), zap.AddCaller(), zap.AddCallerSkip(l.opt.callerSkip)).Sugar()
	if len(l.opt.fields) > 0 {
		zapLog = zapLog.With(CopyFields(l.opt.fields)...)
	}
	if l.opt.namespace != "" {
		zapLog = zapLog.With(zap.Namespace(l.opt.namespace))
	}

	l.lg = zapLog
	return nil
}

// buildConsole build console.
func (l *Logging) buildConsole() []zapcore.Core {
	var (
		sync zapcore.WriteSyncer
		enc  zapcore.Encoder
	)

	if l.opt.encoder.IsConsole() {
		enc = zapcore.NewConsoleEncoder(l.opt.encoderConfig)
	} else {
		enc = zapcore.NewJSONEncoder(l.opt.encoderConfig)
	}

	if l.opt.writer != nil {
		sync = zapcore.AddSync(l.opt.writer)
	} else {
		sync = zapcore.AddSync(WrappedWriteSyncer{os.Stdout})
	}
	return []zapcore.Core{zapcore.NewCore(enc, sync, l.atomicLevel)}
}

// buildCustomWriter build custom writer.
func (l *Logging) buildCustomWriter() []zapcore.Core {
	syncer := l.opt.writer
	if syncer == nil {
		syncer = zapcore.AddSync(WrappedWriteSyncer{os.Stdout})
	}

	var enc zapcore.Encoder
	if l.opt.encoder.IsConsole() {
		enc = zapcore.NewConsoleEncoder(l.opt.encoderConfig)
	} else {
		enc = zapcore.NewJSONEncoder(l.opt.encoderConfig)
	}

	return []zapcore.Core{zapcore.NewCore(enc, zapcore.AddSync(syncer), l.atomicLevel)}
}

// buildFile build rolling file.
func (l *Logging) buildFile() []zapcore.Core {
	_ = l.Sync()
	var enc zapcore.Encoder
	if l.opt.encoder.IsConsole() {
		enc = zapcore.NewConsoleEncoder(l.opt.encoderConfig)
	} else {
		enc = zapcore.NewJSONEncoder(l.opt.encoderConfig)
	}

	syncerRolling := l.createOutput(path.Join(l.opt.path, l.opt.filename))
	l._rollingFiles = append(l._rollingFiles, []io.Writer{syncerRolling}...)
	return []zapcore.Core{zapcore.NewCore(enc, syncerRolling, l.atomicLevel)}
}

// buildFiles build rolling files.
func (l *Logging) buildFiles() []zapcore.Core {
	var (
		cores = make([]zapcore.Core, 0, 5)
		syncerRollingDebug, syncerRollingInfo, syncerRollingWarn,
		syncerRollingError, syncerRollingFatal zapcore.WriteSyncer
	)

	var enc zapcore.Encoder
	if l.opt.encoder.IsConsole() {
		enc = zapcore.NewConsoleEncoder(l.opt.encoderConfig)
	} else {
		enc = zapcore.NewJSONEncoder(l.opt.encoderConfig)
	}

	if err := l.Sync(); err != nil {
		return nil
	}

	syncerRollingDebug = l.createOutput(path.Join(l.opt.path, debugFilename))

	syncerRollingInfo = l.createOutput(path.Join(l.opt.path, infoFilename))

	syncerRollingWarn = l.createOutput(path.Join(l.opt.path, warnFilename))

	syncerRollingError = l.createOutput(path.Join(l.opt.path, errorFilename))

	syncerRollingFatal = l.createOutput(path.Join(l.opt.path, fatalFilename))

	cores = append(cores,
		zapcore.NewCore(enc, syncerRollingDebug, l.LevelEnablerFunc(zap.DebugLevel)),
		zapcore.NewCore(enc, syncerRollingInfo, l.LevelEnablerFunc(zap.InfoLevel)),
		zapcore.NewCore(enc, syncerRollingWarn, l.LevelEnablerFunc(zap.WarnLevel)),
		zapcore.NewCore(enc, syncerRollingError, l.LevelEnablerFunc(zap.ErrorLevel)),
		zapcore.NewCore(enc, syncerRollingFatal, l.LevelEnablerFunc(zap.FatalLevel)),
	)

	l._rollingFiles = append(l._rollingFiles, []io.Writer{syncerRollingDebug, syncerRollingInfo, syncerRollingWarn, syncerRollingError, syncerRollingFatal}...)
	return cores
}

func (l *Logging) createOutput(filename string) zapcore.WriteSyncer {
	return zapcore.AddSync(colorable.NewNonColorable(NewRollingFile(
		filename,
		HourlyRolling,
		l.opt.maxSize,
		l.opt.maxBackups,
		l.opt.maxAge,
		l.opt.localTime,
		l.opt.compress,
	)))
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

func (l *Logging) WithFields(fields map[string]any) Logger {
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
	l.opt.level = lv
	l.atomicLevel.SetLevel(lv.unmarshalZapLevel())
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
	if l.lg == nil {
		return nil
	}

	for _, w := range l._rollingFiles {
		r, ok := w.(*RollingFile)
		if ok {
			r.Close()
		}
	}

	return l.lg.Sync()
}

// WithCallDepth returns a shallow copy of l with its caller skip
func WithCallDepth(callDepth int) Logger {
	return DefaultLogger.WithCallDepth(callDepth)
}

// WithContext returns a shallow copy of l with its context changed
// to ctx. The provided ctx must be non-nil.
func WithContext(ctx context.Context) Logger {
	return DefaultLogger.WithContext(ctx)
}

// WithFields is a helper to create a []interface{} of key-value pairs.
func WithFields(fields map[string]interface{}) Logger {
	return DefaultLogger.WithFields(fields)
}

// SetLevel set logger level
func SetLevel(lv Level) {
	DefaultLogger.SetLevel(lv)
}

func Debug(args ...interface{}) {
	DefaultLogger.WithCallDepth(callerSkipOffset).Debug(args...)
}

func Info(args ...interface{}) {
	DefaultLogger.WithCallDepth(callerSkipOffset).Info(args...)
}

func Warn(args ...interface{}) {
	DefaultLogger.WithCallDepth(callerSkipOffset).Warn(args...)
}

func Error(args ...interface{}) {
	DefaultLogger.WithCallDepth(callerSkipOffset).Error(args...)
}

func Fatal(args ...interface{}) {
	DefaultLogger.WithCallDepth(callerSkipOffset).Fatal(args...)
}

func Debugf(template string, args ...interface{}) {
	DefaultLogger.WithCallDepth(callerSkipOffset).Debugf(template, args...)
}

func Infof(template string, args ...interface{}) {
	DefaultLogger.WithCallDepth(callerSkipOffset).Infof(template, args...)
}

func Warnf(template string, args ...interface{}) {
	DefaultLogger.WithCallDepth(callerSkipOffset).Warnf(template, args...)
}

func Errorf(template string, args ...interface{}) {
	DefaultLogger.WithCallDepth(callerSkipOffset).Errorf(template, args...)
}

func Fatalf(template string, args ...interface{}) {
	DefaultLogger.WithCallDepth(callerSkipOffset).Fatalf(template, args...)
}

func Debugw(msg string, keysAndValues ...interface{}) {
	DefaultLogger.WithCallDepth(callerSkipOffset).Debugw(msg, keysAndValues...)
}

func Infow(msg string, keysAndValues ...interface{}) {
	DefaultLogger.WithCallDepth(callerSkipOffset).Infow(msg, keysAndValues...)
}

func Warnw(msg string, keysAndValues ...interface{}) {
	DefaultLogger.WithCallDepth(callerSkipOffset).Warnw(msg, keysAndValues...)
}

func Errorw(msg string, keysAndValues ...interface{}) {
	DefaultLogger.WithCallDepth(callerSkipOffset).Errorw(msg, keysAndValues...)
}

func Fatalw(msg string, keysAndValues ...interface{}) {
	DefaultLogger.WithCallDepth(callerSkipOffset).Fatalw(msg, keysAndValues...)
}

func Sync() error {
	return DefaultLogger.Sync()
}
