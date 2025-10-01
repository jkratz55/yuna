package log

import (
	"log/slog"
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

	Error = func(err error) slog.Attr {
		return slog.String("err", err.Error())
	}
)

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
