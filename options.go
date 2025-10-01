package yuna

import (
	"context"
	"net"
	"net/http"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"

	"github.com/jkratz55/yuna/log"
)

type config struct {

	// HTTP settings
	httpPort                int
	readTimeout             time.Duration
	readHeaderTimeout       time.Duration
	writeTimeout            time.Duration
	idleTimeout             time.Duration
	baseContext             func(net.Listener) context.Context
	notFoundHandler         http.Handler
	methodNotAllowedHandler http.Handler

	// Logging
	logger *log.Logger

	// Operations HTTP settings
	operationHTTPPort   int
	metricsEnabled      bool
	pprofEnabled        bool
	healthcheckEnabled  bool
	healthcheckBasePath string

	// Metric/Instrumentation settings
	requestDurationBuckets []float64

	// OpenTelemetry settings,
	traceProvider trace.TracerProvider
	meterProvider metric.MeterProvider
}

func newConfig(opts ...baseOption) *config {
	conf := &config{
		httpPort:                8080,
		readTimeout:             0,
		readHeaderTimeout:       0,
		writeTimeout:            0,
		idleTimeout:             0,
		baseContext:             nil,
		operationHTTPPort:       8082,
		requestDurationBuckets:  []float64{0.010, 0.025, 0.050, 0.100, 0.250, 0.500, 1},
		pprofEnabled:            false,
		metricsEnabled:          false,
		traceProvider:           otel.GetTracerProvider(),
		meterProvider:           otel.GetMeterProvider(),
		healthcheckEnabled:      false,
		healthcheckBasePath:     "/healthz",
		notFoundHandler:         wrapFn(notFound),
		methodNotAllowedHandler: wrapFn(methodNotAllowed),
		logger:                  log.GetLogger(),
	}

	for _, opt := range opts {
		opt.apply(conf)
	}
	return conf
}

type baseOption interface {
	apply(conf *config)
}

type Option interface {
	baseOption
	server()
	client()
}

type option func(conf *config)

func (fn option) server() {}

func (fn option) client() {}

func (fn option) apply(conf *config) {
	fn(conf)
}

type ServerOption interface {
	baseOption
	server()
}

type serverOption func(conf *config)

var _ ServerOption = serverOption(nil)

func (s serverOption) apply(conf *config) {
	s(conf)
}

func (s serverOption) server() {}

type ClientOption interface {
	baseOption
	client()
}

type clientOption func(conf *config)

var _ ClientOption = (*clientOption)(nil)

func (c clientOption) apply(conf *config) {
	c(conf)
}

func (c clientOption) client() {}

// ------------------------------------------------------------------------------------------------
// Common Options
// ------------------------------------------------------------------------------------------------

func WithTraceProvider(tp trace.TracerProvider) Option {
	return option(func(c *config) {
		c.traceProvider = tp
	})
}

func WithMeterProvider(mp metric.MeterProvider) Option {
	return option(func(c *config) {
		c.meterProvider = mp
	})
}

func WithRequestDurationBuckets(buckets []float64) Option {
	return option(func(c *config) {
		c.requestDurationBuckets = buckets
	})
}

// ------------------------------------------------------------------------------------------------
// Server Options
// ------------------------------------------------------------------------------------------------

func WithHTTPPort(port int) ServerOption {
	return serverOption(func(c *config) {
		c.httpPort = port
	})
}

func WithReadTimeout(timeout time.Duration) ServerOption {
	return serverOption(func(c *config) {
		c.readTimeout = timeout
	})
}

func WithReadHeaderTimeout(timeout time.Duration) ServerOption {
	return serverOption(func(c *config) {
		c.readHeaderTimeout = timeout
	})
}

func WithWriteTimeout(timeout time.Duration) ServerOption {
	return serverOption(func(c *config) {
		c.writeTimeout = timeout
	})
}

func WithIdleTimeout(timeout time.Duration) ServerOption {
	return serverOption(func(c *config) {
		c.idleTimeout = timeout
	})
}

func WithBaseContext(fn func(net.Listener) context.Context) ServerOption {
	return serverOption(func(c *config) {
		c.baseContext = fn
	})
}

func WithNotFoundHandler(handler http.Handler) ServerOption {
	return serverOption(func(c *config) {
		c.notFoundHandler = handler
	})
}

func WithMethodNotAllowedHandler(handler http.Handler) ServerOption {
	return serverOption(func(c *config) {
		c.methodNotAllowedHandler = handler
	})
}

func WithOperationsHttpPort(port int) ServerOption {
	return serverOption(func(c *config) {
		c.operationHTTPPort = port
	})
}

func WithMetrics() ServerOption {
	return serverOption(func(c *config) {
		c.metricsEnabled = true
	})
}

func WithPPROF() ServerOption {
	return serverOption(func(c *config) {
		c.pprofEnabled = true
	})
}

func WithHealthChecks() ServerOption {
	return serverOption(func(c *config) {
		c.healthcheckEnabled = true
	})
}

func WithHealthChecksBasePath(basePath string) ServerOption {
	return serverOption(func(c *config) {
		c.healthcheckBasePath = basePath
	})
}

// WithLogger sets the Logger used by Yuna, and the request scoped Logger from calling log.LoggerFromCtx
func WithLogger(logger *log.Logger) ServerOption {
	return serverOption(func(c *config) {
		c.logger = logger
	})
}
