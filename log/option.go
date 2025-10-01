package log

import (
	"io"
	"os"
)

type config struct {
	writer        io.Writer
	level         Level
	includeSource bool
	replaceAttr   ReplaceAttrFunc
}

func newConfig() *config {
	return &config{
		writer:        os.Stderr,
		level:         DefaultLogLevel,
		includeSource: false,
		replaceAttr:   levelAttrFormatter,
	}
}

type Option func(*config)

func WithWriter(w io.Writer) Option {
	return func(c *config) {
		c.writer = w
	}
}

func WithLevel(lvl Level) Option {
	return func(c *config) {
		c.level = lvl
	}
}

func WithSource() Option {
	return func(c *config) {
		c.includeSource = true
	}
}

func WithReplaceAttr(fn ReplaceAttrFunc) Option {
	return func(c *config) {
		c.replaceAttr = fn
	}
}
