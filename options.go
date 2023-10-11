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
	debugFilename = "debug"
	infoFilename  = "info"
	warnFilename  = "warn"
	errorFilename = "error"
	fatalFilename = "fatal"
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

	fileOptions
}

type fileOptions struct {
	// mode is the logging mode. default is `consoleMode`
	mode string
	// basePath is the base path of log file. default is `""`
	path string
	// filename is the log filename. default is `""`
	filename string
	// maxAge is the maximum number of days to retain old log files based on the
	maxAge int
	// maxSize is the maximum size in megabytes of the log file before it gets rotated.
	maxSize int
	// maxBackups is the maximum number of old log files to retain.
	maxBackups int
	// localTime is the time zone to use when displaying timestamps.
	localTime bool
	// compress is the compression type for old logs. disabled by default.
	compress bool
	// compress is the rolling format for old logs. default is `HourlyRolling`
	roll RollingFormat
	// writer is the writer of logger.
	writer io.Writer
}

func newOptions(opts ...Option) Options {
	opt := Options{
		level: InfoLevel,
		fileOptions: fileOptions{
			mode: ConsoleMode,
			path: "./logs",
		},
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

// WithMaxAge Setter function to set the maximum log age.
func WithMaxAge(maxAge int) Option {
	return func(o *Options) {
		o.maxAge = maxAge
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

// WithLocalTime Setter function to set the local time option.
func WithLocalTime(localTime bool) Option {
	return func(o *Options) {
		o.localTime = localTime
	}
}

// WithCompress Setter function to set the compress option.
func WithCompress(compress bool) Option {
	return func(o *Options) {
		o.compress = compress
	}
}

// WithRoll Setter function to set the rolling format.
func WithRoll(roll RollingFormat) Option {
	return func(o *Options) {
		o.roll = roll
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

func WithWriter(w io.Writer) Option {
	return func(o *Options) {
		o.writer = w
	}
}
