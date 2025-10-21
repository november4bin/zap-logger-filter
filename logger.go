package zaploggerfilter

import (
	"os"
	"sync"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

type ZapCoreType string

const (
	Console ZapCoreType = "console"
	File    ZapCoreType = "file"
)

// Config 配置
type Config struct {
	// 类型
	Type ZapCoreType
	// 名称
	// 用于指定日志记录器输出日志
	Name string
	// 日志级别
	Level string
	// 是否启动过滤
	SensitiveFilter bool
	// 过滤字段
	SensitiveFields []string
	// 日志文件存储路径
	Path string
	// 单个日志文件最大尺寸
	MaxSize int
	// 最大时间
	MaxAge int
	// 最多保留个数
	MaxBackups int
	// 是否压缩
	Compress bool
}

var (
	// L 全局日志记录器
	L *zap.Logger
	// 日志记录器列表
	l sync.Map
	// encoder 默认编码器
	encoder = zapcore.NewJSONEncoder(zapcore.EncoderConfig{
		TimeKey:       "time",
		LevelKey:      "level",
		NameKey:       "logger",
		CallerKey:     "caller",
		MessageKey:    "msg",
		StacktraceKey: "stacktrace",
		LineEnding:    zapcore.DefaultLineEnding,
		EncodeLevel:   zapcore.LowercaseLevelEncoder,
		EncodeTime: func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
			enc.AppendString(t.Format(time.RFC3339))
		},
		EncodeDuration: zapcore.MillisDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	})
	once sync.Once
)

// Init 初始化
func Init(cfg []Config) {
	once.Do(func() {
		cores := make([]zapcore.Core, 0, len(cfg))

		if len(cfg) > 0 {
			for _, c := range cfg {
				core := newCore(c)
				cores = append(cores, core)
				l.Store(c.Name, newLogger(core))
			}
		} else {
			cores = append(cores, zapcore.NewCore(encoder, zapcore.AddSync(os.Stdout), zapcore.DebugLevel))
		}

		L = newLogger(zapcore.NewTee(cores...))

	})
}

func newCore(cfg Config) zapcore.Core {
	var enc zapcore.Encoder
	if cfg.SensitiveFilter {
		enc = &SensitiveDataEncoder{
			Encoder: encoder,
			Filter:  NewSensitiveDataFilter(cfg.SensitiveFields),
		}
	} else {
		enc = encoder
	}

	var w zapcore.WriteSyncer
	var core zapcore.Core
	switch cfg.Type {
	case Console:
		w = zapcore.AddSync(os.Stdout)
		core = zapcore.NewCore(encoder, zapcore.AddSync(os.Stdout), getLoggerLevel(cfg.Level))
	case File:
		w = zapcore.AddSync(&lumberjack.Logger{
			Filename:   cfg.Path,
			MaxSize:    cfg.MaxSize,
			MaxBackups: cfg.MaxBackups,
			MaxAge:     cfg.MaxAge,
			Compress:   cfg.Compress,
		})
	}

	core = zapcore.NewCore(
		enc,
		w,
		getLoggerLevel(cfg.Level),
	)
	return core
}

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

func newLogger(core zapcore.Core, options ...zap.Option) *zap.Logger {
	options = append(options, zap.AddCaller())
	return zap.New(core, options...)
}

func AddTagetLogger(c Config) {
	core := newCore(c)

	l.Store(c.Name, newLogger(core))
}

func DebugTo(taget string, msg string, fields ...zapcore.Field) {
	LogTo(taget, zapcore.DebugLevel, msg, fields...)
}

func InfoTo(taget string, msg string, fields ...zapcore.Field) {
	LogTo(taget, zapcore.InfoLevel, msg, fields...)
}

func WarnTo(taget string, msg string, fields ...zapcore.Field) {
	LogTo(taget, zapcore.WarnLevel, msg, fields...)
}

func ErrorTo(taget string, msg string, fields ...zapcore.Field) {
	LogTo(taget, zapcore.ErrorLevel, msg, fields...)
}

func LogTo(taget string, lvl zapcore.Level, msg string, fields ...zapcore.Field) {
	v, ok := l.Load(taget)
	if ok {
		v.(*zap.Logger).Log(lvl, msg, fields...)
	}
}

func Sync() {
	_ = L.Sync()

	l.Range(func(_, v interface{}) bool {
		_ = v.(*zap.Logger).Sync()
		return true
	})
}
