package log

import (
	"fmt"
	"log/slog"
	"strings"
)

// A Level is the importance or severity of a log event. The higher the level, the more important or severe the event.
type Level = slog.Level

const (
	LevelTrace = Level(-5)
	LevelDebug = slog.LevelDebug
	LevelInfo  = slog.LevelInfo
	LevelWarn  = slog.LevelWarn
	LevelError = slog.LevelError
	LevelPanic = Level(10)
	LevelFatal = Level(12)
)

// DefaultLogLevel is the default log level
const DefaultLogLevel = LevelInfo

func ParseLevel(lvl string) (Level, error) {
	switch strings.ToLower(lvl) {
	case "trace":
		return LevelTrace, nil
	case "debug":
		return LevelDebug, nil
	case "info":
		return LevelInfo, nil
	case "warn":
		return LevelWarn, nil
	case "error":
		return LevelError, nil
	case "panic":
		return LevelPanic, nil
	case "fatal":
		return LevelFatal, nil
	default:
		return LevelInfo, fmt.Errorf("invalid log level: %s", lvl)
	}
}

func levelAttrFormatter(_ []string, a slog.Attr) slog.Attr {
	if a.Key == slog.LevelKey {
		if lv, ok := a.Value.Any().(slog.Level); ok {
			switch lv {
			case LevelTrace:
				a.Value = slog.StringValue("TRACE")
			case LevelDebug:
				a.Value = slog.StringValue("DEBUG")
			case LevelInfo:
				a.Value = slog.StringValue("INFO")
			case LevelWarn:
				a.Value = slog.StringValue("WARN")
			case LevelError:
				a.Value = slog.StringValue("ERROR")
			case LevelPanic:
				a.Value = slog.StringValue("PANIC")
			case LevelFatal:
				a.Value = slog.StringValue("FATAL")
			default:
				// Fallback to slog's default string if an unknown custom level is used.
				a.Value = slog.StringValue(lv.String())
			}
		}
	}
	return a
}
