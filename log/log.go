package log

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strconv"

	"github.com/jkratz55/yuna/internal"
)

var (
	globalLogger *Logger
)

func init() {
	var lvl slog.Level
	envLogLevel, ok := os.LookupEnv("YUNA_LOG_LEVEL")
	if !ok {
		lvl = DefaultLogLevel
	} else {
		var err error
		lvl, err = ParseLevel(envLogLevel)
		if err != nil {
			fmt.Printf("invalid log level: %s, using default: %s\n", envLogLevel, DefaultLogLevel)
			lvl = DefaultLogLevel
		}
	}

	includeSource, _ := strconv.ParseBool(os.Getenv("YUNA_LOG_INCLUDE_SOURCE"))
	leveler := new(slog.LevelVar)
	leveler.Set(lvl)

	handler := slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		AddSource:   includeSource,
		Level:       leveler,
		ReplaceAttr: levelAttrFormatter,
	})
	slogLogger := slog.New(handler).With("logger", "yuna")

	globalLogger = &Logger{
		Logger: slogLogger,
		level:  leveler,
	}
}

func GetLogger() *Logger {
	return globalLogger
}

func LoggerFromCtx(ctx context.Context) *Logger {
	logger, ok := ctx.Value(internal.ContextKeyLogger).(*Logger)
	if !ok || logger == nil {
		return globalLogger
	}
	return logger
}

func With(args ...interface{}) *Logger {
	globalLogger = globalLogger.With(args...)
	return globalLogger
}

type Logger struct {
	*slog.Logger
	level *slog.LevelVar
}

func New(opts ...Option) *Logger {

	conf := newConfig()
	for _, opt := range opts {
		opt(conf)
	}

	leveler := new(slog.LevelVar)
	leveler.Set(conf.level)
	handler := slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		AddSource:   conf.includeSource,
		Level:       leveler,
		ReplaceAttr: ChainReplaceAttr(levelAttrFormatter, conf.replaceAttr),
	})
	slogLogger := slog.New(handler)
	return &Logger{
		Logger: slogLogger,
		level:  leveler,
	}
}

func (l *Logger) Level() slog.Level {
	return l.level.Level()
}

func (l *Logger) SetLevel(lvl slog.Level) {
	l.level.Set(lvl)
}

func (l *Logger) Panic(msg string, args ...interface{}) {
	l.Logger.Log(context.Background(), LevelPanic, msg, args...)
	panic(msg)
}

func (l *Logger) PanicContext(ctx context.Context, msg string, args ...interface{}) {
	l.Logger.Log(ctx, LevelPanic, msg, args...)
	panic(msg)
}

func (l *Logger) Fatal(msg string, args ...interface{}) {
	l.Logger.Log(context.Background(), LevelFatal, msg, args...)
	os.Exit(1)
}

func (l *Logger) FatalContext(ctx context.Context, msg string, args ...interface{}) {
	l.Logger.Log(ctx, LevelFatal, msg, args...)
	os.Exit(1)
}

func (l *Logger) With(args ...interface{}) *Logger {
	return &Logger{
		Logger: l.Logger.With(args...),
		level:  l.level,
	}
}

func (l *Logger) WithGroup(group string) *Logger {
	return &Logger{
		Logger: l.Logger.WithGroup(group),
		level:  l.level,
	}
}
