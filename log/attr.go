package log

import (
	"log/slog"
	"runtime/debug"
	"strconv"
	"strings"
)

var (
	String     = slog.String
	Int64      = slog.Int64
	Int        = slog.Int
	Uint64     = slog.Uint64
	Float64    = slog.Float64
	Bool       = slog.Bool
	Time       = slog.Time
	Duration   = slog.Duration
	Group      = slog.Group
	GroupAttrs = slog.GroupAttrs
	Any        = slog.Any
)

func Error(err error) slog.Attr {
	return slog.String("err", err.Error())
}

func Stack() slog.Attr {
	return slog.String("stacktrace", string(debug.Stack()))
}

func PrettyStack() slog.Attr {
	stack := debug.Stack()
	frames := parseStack(stack, 2)
	return slog.Any("stacktrace", frames)
}

type Frame struct {
	Function string `json:"function"`
	File     string `json:"file"`
	Line     int    `json:"line"`
}

func parseStack(stack []byte, skipFrames int) []Frame {
	lines := strings.Split(string(stack), "\n")
	skipLines := 1
	if skipFrames > 0 {
		skipLines = skipFrames*2 + skipLines
	}

	if len(lines) < skipLines {
		return []Frame{}
	}

	lines = lines[skipLines:]
	frames := make([]Frame, 0, len(lines))

	for i := 0; i < len(lines)-1; i += 2 {
		function := lines[i]
		function = strings.TrimPrefix(function, "created by ")

		fileLine := lines[i+1]
		parts := strings.Split(fileLine, ":")
		if len(parts) != 2 {
			continue
		}
		file := parts[0]
		file = strings.TrimLeft(file, "\t")

		lineNum := -1
		subParts := strings.Split(parts[1], " ")
		if len(subParts) == 0 {
			lineNum, _ = strconv.Atoi(parts[1])
		} else {
			lineNum, _ = strconv.Atoi(subParts[0])
		}

		frames = append(frames, Frame{
			Function: function,
			File:     file,
			Line:     lineNum,
		})
	}

	return frames
}

// ReplaceAttrFunc is the signature of slog's ReplaceAttr for convenience.
type ReplaceAttrFunc func(groups []string, a slog.Attr) slog.Attr

func ChainReplaceAttr(fns ...ReplaceAttrFunc) ReplaceAttrFunc {
	return func(groups []string, a slog.Attr) slog.Attr {
		out := a
		for _, fn := range fns {
			if fn == nil {
				continue
			}
			out = fn(groups, out)
			if out.Key == "" {
				// Dropped; keep propagating the "dropped" attr to respect user intent.
				// Subsequent funcs will see an empty attr and may choose to restore or keep it dropped.
			}
		}
		return out
	}
}
