package yuna

import (
	"context"
	"net"
	"net/http"
	"time"

	"github.com/go-resty/resty/v2"
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

	// Authentication settings
	authenticator HttpAuthenticator

	// Resty specific settings
	onBeforeRequest    func(c *resty.Client, r *resty.Request) error
	onAfterResponse    func(c *resty.Client, r *resty.Response) error
	onClientError      func(r *resty.Request, err error)
	clientInstrumenter HttpClientInstrumenter
}

func newConfig(opts ...baseOption) *config {
	conf := &config{
		httpPort:                8080,
		readTimeout:             0,
		readHeaderTimeout:       0,
		writeTimeout:            0,
		idleTimeout:             0,
		baseContext:             nil,
		notFoundHandler:         wrapFn(notFound),
		methodNotAllowedHandler: wrapFn(methodNotAllowed),
		logger:                  log.GetLogger(),
		operationHTTPPort:       8082,
		metricsEnabled:          false,
		pprofEnabled:            false,
		healthcheckEnabled:      false,
		healthcheckBasePath:     "/healthz",
		requestDurationBuckets:  []float64{0.010, 0.025, 0.050, 0.100, 0.250, 0.500, 1},
		traceProvider:           otel.GetTracerProvider(),
		meterProvider:           otel.GetMeterProvider(),
		authenticator:           nil,
		onBeforeRequest:         func(c *resty.Client, r *resty.Request) error { return nil },
		onAfterResponse:         func(c *resty.Client, r *resty.Response) error { return nil },
		onClientError:           func(r *resty.Request, err error) {},
		clientInstrumenter:      nil,
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

// WithTraceProvider sets the TraceProvider used by Yuna. Defaults to the global TraceProvider.
func WithTraceProvider(tp trace.TracerProvider) Option {
	return option(func(c *config) {
		c.traceProvider = tp
	})
}

// WithMeterProvider sets the MeterProvider used by Yuna. Defaults to the global MeterProvider.
func WithMeterProvider(mp metric.MeterProvider) Option {
	return option(func(c *config) {
		c.meterProvider = mp
	})
}

// WithRequestDurationBuckets sets the buckets for the request duration histogram.
func WithRequestDurationBuckets(buckets []float64) Option {
	return option(func(c *config) {
		c.requestDurationBuckets = buckets
	})
}

// ------------------------------------------------------------------------------------------------
// Server Options
// ------------------------------------------------------------------------------------------------

// WithHTTPPort sets the port for the HTTP server. The default is 8080.
func WithHTTPPort(port int) ServerOption {
	return serverOption(func(c *config) {
		c.httpPort = port
	})
}

// WithReadTimeout sets the server's ReadTimeout.
func WithReadTimeout(timeout time.Duration) ServerOption {
	return serverOption(func(c *config) {
		c.readTimeout = timeout
	})
}

// WithReadHeaderTimeout sets the server's ReadHeaderTimeout.
func WithReadHeaderTimeout(timeout time.Duration) ServerOption {
	return serverOption(func(c *config) {
		c.readHeaderTimeout = timeout
	})
}

// WithWriteTimeout sets the server's WriteTimeout.
func WithWriteTimeout(timeout time.Duration) ServerOption {
	return serverOption(func(c *config) {
		c.writeTimeout = timeout
	})
}

// WithIdleTimeout sets the server's IdleTimeout.
func WithIdleTimeout(timeout time.Duration) ServerOption {
	return serverOption(func(c *config) {
		c.idleTimeout = timeout
	})
}

// WithBaseContext sets a function that is called to create the base context for each request.
func WithBaseContext(fn func(net.Listener) context.Context) ServerOption {
	return serverOption(func(c *config) {
		c.baseContext = fn
	})
}

// WithNotFoundHandler sets the handler for HTTP 404 Not Found.
func WithNotFoundHandler(handler http.Handler) ServerOption {
	return serverOption(func(c *config) {
		c.notFoundHandler = handler
	})
}

// WithMethodNotAllowedHandler sets the handler for HTTP 405 Method Not Allowed.
func WithMethodNotAllowedHandler(handler http.Handler) ServerOption {
	return serverOption(func(c *config) {
		c.methodNotAllowedHandler = handler
	})
}

// WithOperationsHttpPort sets the port for the operational server. The default is 8082.
func WithOperationsHttpPort(port int) ServerOption {
	return serverOption(func(c *config) {
		c.operationHTTPPort = port
	})
}

// WithMetrics enables the Prometheus metrics endpoint on the operational server.
func WithMetrics() ServerOption {
	return serverOption(func(c *config) {
		c.metricsEnabled = true
	})
}

// WithPPROF enables the pprof endpoint on the operational server.
func WithPPROF() ServerOption {
	return serverOption(func(c *config) {
		c.pprofEnabled = true
	})
}

// WithHealthChecks enables health check endpoints on the operational server.
//
// This enables two endpoints by default:
//   - /healthz/live
//   - /healthz/ready
//
// The base path can be modified using WithHealthChecksBasePath.
func WithHealthChecks() ServerOption {
	return serverOption(func(c *config) {
		c.healthcheckEnabled = true
	})
}

// WithHealthChecksBasePath sets the base path for health checks.
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

// WithAuthentication enables authentication for all endpoints/routes registered with Yuna using
// the provided HttpAuthenticator.
//
// WithAuthentication does not enforce the client/user is authenticated, it simply attempts to
// authenticate the client/user, and stores the Principal returned by the HttpAuthenticator in the
// context of the request. The Principal can be retrieved in a Handler using the PrincipalFromCtx
// function.
//
// It is important to note that the HttpAuthenticator is responsible for handling authentication
// failures due to invalid or missing credentials. However, if the HttpAuthenticator returns a
// non-nil error value, the Authenticate middleware will respond with an HTTP 500 InternalServerError.
//
// To protect a specific endpoint/route, or set of routes, use the Authenticated or RequireRole
// middleware.
func WithAuthentication(authenticator HttpAuthenticator) ServerOption {
	if authenticator == nil {
		panic("authenticator cannot be nil")
	}
	return serverOption(func(c *config) {
		c.authenticator = authenticator
	})
}

// ------------------------------------------------------------------------------------------------
// Client Options
// ------------------------------------------------------------------------------------------------

// WithClientOnBeforeRequest sets a Resty hook that is called before a request is sent.
func WithClientOnBeforeRequest(fn func(c *resty.Client, r *resty.Request) error) ClientOption {
	if fn == nil {
		fn = func(c *resty.Client, r *resty.Request) error { return nil }
	}
	return clientOption(func(c *config) {
		c.onBeforeRequest = fn
	})
}

// WithClientOnAfterResponse sets a Resty hook that is called after a response is received.
func WithClientOnAfterResponse(fn func(c *resty.Client, r *resty.Response) error) ClientOption {
	if fn == nil {
		fn = func(c *resty.Client, r *resty.Response) error { return nil }
	}
	return clientOption(func(c *config) {
		c.onAfterResponse = fn
	})
}

// WithClientOnClientError sets a Resty hook that is called when a client error occurs.
func WithClientOnClientError(fn func(r *resty.Request, err error)) ClientOption {
	if fn == nil {
		fn = func(r *resty.Request, err error) {}
	}
	return clientOption(func(c *config) {
		c.onClientError = fn
	})
}

// WithClientHttpInstrumenter sets the instrumenter used by the Resty client.
func WithClientHttpInstrumenter(is HttpClientInstrumenter) ClientOption {
	return clientOption(func(c *config) {
		c.clientInstrumenter = is
	})
}
