package zaploggerfilter

import (
	"os"
	"sync"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

type ZapCoreType string

const (
	Console ZapCoreType = "console"
	File    ZapCoreType = "file"
)

type Config struct {
	Type            ZapCoreType
	Name            string
	Level           string
	SensitiveFilter bool
	SensitiveFields []string
	Path            string
	MaxSize         int
	MaxAge          int
	MaxBackups      int
	Compress        bool
}

var (
	// L 全局日志记录器
	L *zap.Logger
	// l 日志记录器映射
	l sync.Map
	// encoderConfig 日志编码器配置
	encoderConfig = zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.RFC3339TimeEncoder,
		EncodeDuration: zapcore.MillisDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}
	defaultLogLevel = zapcore.DebugLevel
	defaultLogName  = "default"
	once            sync.Once
)

// Init 初始化日志记录器
func Init(cfg []Config) {
	once.Do(func() {
		// 创建默认日志记录器核心
		defaultLogCore := zapcore.NewCore(zapcore.NewConsoleEncoder(encoderConfig), zapcore.AddSync(os.Stdout), defaultLogLevel)
		defaultLog := newLogger(defaultLogCore)
		l.Store(defaultLogName, defaultLog)

		if len(cfg) > 0 {
			// 创建日志记录器核心
			cores := make([]zapcore.Core, 0, len(cfg))
			for _, c := range cfg {
				core := newCore(c)
				cores = append(cores, core)
				l.Store(c.Name, newLogger(core))
			}

			L = newLogger(zapcore.NewTee(cores...))
		} else {
			// 如果没有配置日志记录器，默认使用控制台记录器
			L = defaultLog
		}

	})
}

// newCore 创建日志记录器核心
// 如果日志记录器类型无效，会触发panic
func newCore(cfg Config) zapcore.Core {
	var encoder zapcore.Encoder

	// 根据配置创建日志编码器
	if cfg.SensitiveFilter {
		// 开启敏感数据过滤，使用敏感数据过滤编码器
		encoder = &SensitiveDataEncoder{
			Encoder: encoder,
			Filter:  NewSensitiveDataFilter(cfg.SensitiveFields),
		}
	} else {
		// 未开启敏感数据过滤，根据日志记录器类型创建编码器
		switch cfg.Type {
		case File:
			encoder = zapcore.NewJSONEncoder(encoderConfig)
		case Console:
			encoder = zapcore.NewConsoleEncoder(encoderConfig)
		default:
			panic("unknown zap core type: " + cfg.Type)
		}
	}

	switch cfg.Type {
	case Console:
		return zapcore.NewCore(zapcore.NewConsoleEncoder(encoderConfig), zapcore.AddSync(os.Stdout), getLoggerLevel(cfg.Level))
	case File:
		return zapcore.NewCore(
			encoder,
			zapcore.AddSync(&lumberjack.Logger{
				Filename:   cfg.Path,
				MaxSize:    cfg.MaxSize,
				MaxBackups: cfg.MaxBackups,
				MaxAge:     cfg.MaxAge,
				Compress:   cfg.Compress,
			}),
			getLoggerLevel(cfg.Level),
		)
	default:
		return nil
	}
}

// getLoggerLevel 获取日志级别
// 如果配置的日志级别无效，会触发panic
func getLoggerLevel(level string) zapcore.Level {
	switch level {
	case "debug":
		return zap.DebugLevel
	case "info":
		return zap.InfoLevel
	case "warn":
		return zap.WarnLevel
	case "error":
		return zap.ErrorLevel
	case "panic":
		return zap.PanicLevel
	case "fatal":
		return zap.FatalLevel
	default:
		panic("invalid log level")
	}
}

// newLogger 创建日志记录器
func newLogger(core zapcore.Core, options ...zap.Option) *zap.Logger {
	options = append(options, zap.AddCaller())
	return zap.New(core, options...)
}

// AddTargetLogger 添加目标日志记录器
func AddTargetLogger(c Config) {
	core := newCore(c)

	l.Store(c.Name, newLogger(core))
}

// GetTargetLogger 获取目标日志记录器
func GetTargetLogger(target string) (*zap.Logger, bool) {
	lg, ok := l.Load(target)
	if ok {
		return lg.(*zap.Logger), true
	}
	return nil, false
}

// DebugTo 向指定目标记录调试级别的日志
func DebugTo(target string, msg string, fields ...zapcore.Field) {
	LogTo(target, zapcore.DebugLevel, msg, fields...)
}

// InfoTo 向指定目标记录信息级别的日志
func InfoTo(target string, msg string, fields ...zapcore.Field) {
	LogTo(target, zapcore.InfoLevel, msg, fields...)
}

// WarnTo 向指定目标记录警告级别的日志
func WarnTo(target string, msg string, fields ...zapcore.Field) {
	LogTo(target, zapcore.WarnLevel, msg, fields...)
}

// ErrorTo 向指定目标记录错误级别的日志
func ErrorTo(target string, msg string, fields ...zapcore.Field) {
	LogTo(target, zapcore.ErrorLevel, msg, fields...)
}

// LogTo 向指定目标记录日志
func LogTo(target string, lvl zapcore.Level, msg string, fields ...zapcore.Field) {
	v, ok := l.Load(target)
	if ok {
		v.(*zap.Logger).Log(lvl, msg, fields...)
	}
}

// Sync 同步日志记录器
func Sync() {
	_ = L.Sync()

	l.Range(func(_, v interface{}) bool {
		_ = v.(*zap.Logger).Sync()
		return true
	})
}
