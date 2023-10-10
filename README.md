# go 日志模块

> 基础日志模块，基于 `uber.zap` 封装

## 日志级别

```go
// DebugLevel level. Usually only enabled when debugging. Very verbose logging.
DebugLevel = iota + 1
// InfoLevel is the default logging priority.
// General operational entries about what's going on inside the application.
InfoLevel
// WarnLevel level. Non-critical entries that deserve eyes.
WarnLevel
// ErrorLevel level. Logs. Used for errors that should definitely be noted.
ErrorLevel
// FatalLevel level. Logs and then calls `logger.Exit(1)`. highest level of severity.
FatalLevel
```

低于级别的日志不会输出，高于级别的日志会输出到对应的文件，建议 `开发环境` 日志级别设置为 `DEBUG`, `线上环境` 日志级别设置为 `INFO`

## 打印日志方法

```go
// Logger is the interface for Logger types
type Logger interface {
	// SetLevel set logger level
	SetLevel(lv Level)
	// WithContext with context
	WithContext(ctx context.Context) Logger
	// WithFields set fields to always be logged
	WithFields(fields map[string]any) Logger
	// WithCallDepth  with logger call depth.
	WithCallDepth(callDepth int) Logger
	// Debug uses fmt.Sprint to construct and log a message.
	Debug(args ...interface{})
	// Info uses fmt.Sprint to construct and log a message.
	Info(args ...interface{})
	// Warn uses fmt.Sprint to construct and log a message.
	Warn(args ...interface{})
	// Error uses fmt.Sprint to construct and log a message.
	Error(args ...interface{})
	// Fatal uses fmt.Sprint to construct and log a message, then calls os.Exit.
	Fatal(args ...interface{})
	// Debugf uses fmt.Sprintf to log a templated message.
	Debugf(template string, args ...interface{})
	// Infof uses fmt.Sprintf to log a templated message.
	Infof(template string, args ...interface{})
	// Warnf uses fmt.Sprintf to log a templated message.
	Warnf(template string, args ...interface{})
	// Errorf uses fmt.Sprintf to log a templated message.
	Errorf(template string, args ...interface{})
	// Fatalf uses fmt.Sprintf to log a templated message, then calls os.Exit.
	Fatalf(template string, args ...interface{})
	// Debugw logs a message with some additional context. The variadic key-value
	// pairs are treated as they are in With.
	//
	// When debug-level logging is disabled, this is much faster than
	//  s.With(keysAndValues).Debug(msg)
	Debugw(msg string, keysAndValues ...interface{})
	// Infow logs a message with some additional context. The variadic key-value
	// pairs are treated as they are in With.
	Infow(msg string, keysAndValues ...interface{})
	// Warnw logs a message with some additional context. The variadic key-value
	// pairs are treated as they are in With.
	Warnw(msg string, keysAndValues ...interface{})
	// Errorw logs a message with some additional context. The variadic key-value
	// pairs are treated as they are in With.
	Errorw(msg string, keysAndValues ...interface{})
	// Fatalw logs a message with some additional context, then calls os.Exit. The
	// variadic key-value pairs are treated as they are in With.
	Fatalw(msg string, keysAndValues ...interface{})
	// Sync logger sync
	Sync() error
}
```

## 简单使用

> 使用日志的时候注意尽量避免使用 `Fatal` 级别的日志，虽然提供了，但是不建议使用，使用 `Fatal` 记录消息后，直接调用 os.Exit(1)，这意味着： 在其他 goroutine defer 语句不会被执行； 各种 buffers 不会被 flush，包括日志的； 临时文件或者目录不会被移除； 不要使用 fatal 记录日志，而是向调用者返回错误。如果错误一直持续到 main.main。main.main 那就是在退出之前做处理任何清理操作的正确位置。

### 日志初始化
```go
logger.InitLogger(New(
    WithPath("../logs"),
    WithLevel(DebugLevel),
    WithMode(ConsoleMode),
    Fields(map[string]interface{}{
    "app_id":      "nextmicro",
    "instance_id": "JeffreyBool",
    }),
))
```

上面会覆盖日志默认的行为，因为方便使用，默认调用就会初始化

### 各种级别日志输出
```go
func TestDebug(t *testing.T) {
	logger.Debug("test debug logger")
	logger.Debugf("debug test time:%d", time.Now().Unix())
	logger.Sync()
}

func TestInfo(t *testing.T) {
	logger.Info("测试日志")
	logger.Infof("name:%s, age:%d", "pandatv", 14)
	logger.Infow("我们都是中国人", "age", 18, "name", "zhanggaoyuan")
    logger.Sync()
}

func TestWarn(t *testing.T) {
	logger.Warn("test warn logger")
	logger.Warnf("warn test time:%d", time.Now().Unix())
    logger.Sync()
}

func TestError(t *testing.T) {
	logger.Error("test error logger")
	logger.Errorf("error test time:%d", time.Now().Unix())
    logger.Sync()
}
```

### 日志多实例使用
> 多实例日志必须设置 `WithFilename(xxx)` 和普通日志区分，这种日志不需要指定级别，全部会输出到一个文件中
```go
slow := New(
    WithPath("../logs"),
    WithMode(FileMode),
    WithFilename("slow"),
    WithFields(map[string]interface{}{
    "app_id":      "mt",
    "instance_id": "JeffreyBool",
    }),
)

slow.Info(msg)
```
> 需要外部自己保存日志的对象信息

## 注意事项

- 不要使用 `Fatal` 级别日志
- 日志使用完毕后一定要显式调用 `Sync()` 函数将日志落盘