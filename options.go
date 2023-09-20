package logger

import (
	"go.uber.org/zap/zapcore"
)

const (
	spanKey  = "span_id"
	traceKey = "trace_id"

	callerSkipOffset = 2

	fileMode    = "file"
	volumeMode  = "volume"
	consoleMode = "console"
)

type Level zapcore.Level

const (
	// DebugLevel logs are typically voluminous, and are usually disabled in
	// production.
	DebugLevel Level = iota - 1
	// InfoLevel is the default logging priority.
	InfoLevel
	// WarnLevel logs are more important than Info, but don't need individual
	// human review.
	WarnLevel
	// ErrorLevel logs are high-priority. If an application is running smoothly,
	// it shouldn't generate any error-level logs.
	ErrorLevel
	// DPanicLevel logs are particularly important errors. In development the
	// logger panics after writing the message.
	DPanicLevel
	// PanicLevel logs a message, then panics.
	PanicLevel
	// FatalLevel logs a message, then calls os.Exit(1).
	FatalLevel

	_minLevel = DebugLevel
	_maxLevel = FatalLevel

	// InvalidLevel is an invalid value for Level.
	//
	// Core implementations may panic if they see messages of this level.
	InvalidLevel = _maxLevel + 1
)

func (l Level) Level() zapcore.Level {
	return zapcore.Level(l)
}

type Option func(o *Options)

type Options struct {
	// The logging level the Logger should log at. default is `InfoLevel`
	level      Level
	mode       string
	filename   string
	maxAge     int
	maxSize    int
	maxBackups int
	localTime  bool
	compress   bool
	// callerSkip is the number of stack frames to ascend when logging caller info.
	callerSkip int
	// namespace is the namespace of logger.
	namespace string
	// fields is the fields of logger.
	fields map[string]interface{}
	// encoder is the encoder of logger.
	encoder Encoder
	// encoderConfig is the encoder config of logger.
	encoderConfig zapcore.EncoderConfig
}

func newOptions(opts ...Option) Options {
	opt := Options{
		level:      Level(zapcore.InfoLevel),
		mode:       "console",
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
		fields:  make(map[string]interface{}),
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
