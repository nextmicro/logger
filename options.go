package logger

import (
	"io"

	"go.uber.org/zap/zapcore"
)

const (
	spanKey  = "span_id"
	traceKey = "trace_id"

	callerSkipOffset = 1

	FileMode    = "file"
	ConsoleMode = "console"
)

const (
	debugFilename = "debug.log"
	infoFilename  = "info.log"
	warnFilename  = "warn.log"
	errorFilename = "error.log"
	fatalFilename = "fatal.log"
)

type Option func(o *Options)

type Options struct {
	// The logging level the Logger should log at. default is `InfoLevel`
	level Level
	// callerSkip is the number of stack frames to ascend when logging caller info.
	callerSkip int
	// namespace is the namespace of logger.
	namespace string
	// fields is the fields of logger.
	fields map[string]any
	// encoder is the encoder of logger.
	encoder Encoder
	// encoderConfig is the encoder config of logger.
	encoderConfig zapcore.EncoderConfig

	// mode is the logging mode. default is `consoleMode`
	mode string
	// path represents the log file path, default is `logs`.
	path string
	// filename is the log filename. default is `""`
	filename string
	// maxSize represents how much space the writing log file takes up. 0 means no limit. The unit is `MB`.
	// Only take effect when RotationRuleType is `size`
	maxSize int
	// keepDays represents how many days the log files will be kept. Default to keep all files.
	// Only take effect when Mode is `file` or `volume`, both work when Rotation is `daily` or `size`.
	keepDays int
	// keepHours represents how many hours the log files will be kept. Default to keep all files.
	// Only take effect when Mode is `file` or `volume`, both work when Rotation is `daily` or `size`.
	keepHours int
	// maxBackups represents how many backup log files will be kept. 0 means all files will be kept forever.
	// Only take effect when RotationRuleType is `size`.
	// Even though `MaxBackups` sets 0, log files will still be removed
	// if the `KeepDays` limitation is reached.
	maxBackups int
	// compress is the compression type for old logs. disabled by default.
	compress bool
	// rotation represents the type of log rotation rule. Default is `daily`.
	// daily: daily rotation.
	// size: size limited rotation.
	rotation string
	// writer is the writer of logger.
	writer io.Writer
}

func newOptions(opts ...Option) Options {
	opt := Options{
		level:      InfoLevel,
		mode:       ConsoleMode,
		path:       "./logs",
		callerSkip: callerSkipOffset,
		encoderConfig: zapcore.EncoderConfig{
			TimeKey:        "ts",
			MessageKey:     "msg",
			LevelKey:       "level",
			CallerKey:      "caller",
			StacktraceKey:  "stack",
			LineEnding:     zapcore.DefaultLineEnding,
			NameKey:        "Logger",
			EncodeCaller:   zapcore.ShortCallerEncoder,
			EncodeLevel:    zapcore.LowercaseLevelEncoder,
			EncodeTime:     zapcore.ISO8601TimeEncoder, // 日期格式改为"ISO8601"，例如："2020-12-16T19:12:48.771+0800"
			EncodeDuration: zapcore.StringDurationEncoder,
			EncodeName:     zapcore.FullNameEncoder,
		},
		fields:  make(map[string]any),
		encoder: JsonEncoder,
	}

	for _, o := range opts {
		o(&opt)
	}

	return opt
}

type Encoder string

func (e Encoder) String() string {
	return string(e)
}

// IsJson Whether json encoder.
func (e Encoder) IsJson() bool {
	return e.String() == JsonEncoder.String()
}

// IsConsole Whether console encoder.
func (e Encoder) IsConsole() bool {
	return e.String() == ConsoleEncoder.String()
}

const (
	JsonEncoder    Encoder = "json"
	ConsoleEncoder Encoder = "console"
)

// WithLevel Setter function to set the logging level.
func WithLevel(level Level) Option {
	return func(o *Options) {
		o.level = level
	}
}

// WithMode Setter function to set the logging mode.
func WithMode(mode string) Option {
	return func(o *Options) {
		o.mode = mode
	}
}

// WithPath Setter function to set the log path.
func WithPath(path string) Option {
	return func(o *Options) {
		o.path = path
	}
}

// WithFilename Setter function to set the log filename.
func WithFilename(filename string) Option {
	return func(o *Options) {
		o.filename = filename
	}
}

// WithMaxSize Setter function to set the maximum log size.
func WithMaxSize(maxSize int) Option {
	return func(o *Options) {
		o.maxSize = maxSize
	}
}

// WithMaxBackups Setter function to set the maximum number of log backups.
func WithMaxBackups(maxBackups int) Option {
	return func(o *Options) {
		o.maxBackups = maxBackups
	}
}

// WithCompress Setter function to set the compress option.
func WithCompress(compress bool) Option {
	return func(o *Options) {
		o.compress = compress
	}
}

// WithCallerSkip Setter function to set the caller skip value.
func WithCallerSkip(callerSkip int) Option {
	return func(o *Options) {
		o.callerSkip = callerSkip
	}
}

// WithNamespace Setter function to set the namespace.
func WithNamespace(namespace string) Option {
	return func(o *Options) {
		o.namespace = namespace
	}
}

// Fields Setter function to set the logger fields.
func Fields(fields map[string]any) Option {
	return func(o *Options) {
		o.fields = fields
	}
}

// WithEncoder Setter function to set the encoder.
func WithEncoder(encoder Encoder) Option {
	return func(o *Options) {
		o.encoder = encoder
	}
}

// WithEncoderConfig Setter function to set the encoder config.
func WithEncoderConfig(encoderConfig zapcore.EncoderConfig) Option {
	return func(o *Options) {
		o.encoderConfig = encoderConfig
	}
}

func WithKeepHours(keepHours int) Option {
	return func(o *Options) {
		o.keepHours = keepHours
	}
}

func WithKeepDays(keepDays int) Option {
	return func(o *Options) {
		o.keepDays = keepDays
	}
}

func WithRotation(rotation string) Option {
	return func(o *Options) {
		o.rotation = rotation
	}
}

func WithWriter(w io.Writer) Option {
	return func(o *Options) {
		o.writer = w
	}
}
