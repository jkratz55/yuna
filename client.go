package yuna

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/go-resty/resty/v2"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/metric/noop"

	"github.com/jkratz55/yuna/internal"
	"github.com/jkratz55/yuna/internal/metrics"
)

var (
	defaultTransport  http.RoundTripper
	defaultClient     *resty.Client
	clientInitializer sync.Once
)

func init() {
	// Overrides the default http.Transport from the standard library since it has some questionable
	// defaults. In particular, it has a default MaxIdleConnsPerHost of 2, which is too low for a
	// microservice architecture, where high rates of requests to a single host are expected, leading
	// to not being able to make use of connection pooling.
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.MaxIdleConnsPerHost = 100 // Overrides the default of 2
	transport.MaxIdleConns = 250        // Overrides the default of 100
	transport.MaxConnsPerHost = 0       // Enforce no limit on the number of connections

	defaultTransport = transport
}

// DefaultClient returns a default configured [resty.Client].
//
// The client is configured with an http.Transport tuned for high volumes of HTTP requests to a small
// number of hosts, which is common in a microservice architecture. The client is also configured with
// tracing and metrics through OpenTelemetry. The TracerProvider and MeterProvider are taken from the
// global OpenTelemetry configuration.
//
// DefaultClient returns a shared instance of [resty.Client]. Because the reference is shared, modifying
// the client level settings/configuration can/will affect other parts of the application using
// DefaultClient.
//
// To customize the behavior and/or configuration of the client, use [NewClient] instead.
func DefaultClient() *resty.Client {
	// Delayed initialization so that the user has an opportunity to initialize OpenTelemetry before
	// the default client is initialized.
	clientInitializer.Do(func() {
		defaultClient = NewClient()
	})
	return defaultClient
}

// NewClient returns a new [resty.Client] with the specified options.
//
// The client is configured with an http.Transport tuned for high volumes of HTTP requests to a small
// number of hosts, which is common in a microservice architecture.
func NewClient(opts ...ClientOption) *resty.Client {

	baseOpts := make([]baseOption, len(opts))
	for i, opt := range opts {
		baseOpts[i] = opt
	}

	conf := newConfig(baseOpts...)
	transport := otelhttp.NewTransport(defaultTransport,
		otelhttp.WithTracerProvider(conf.traceProvider),
		otelhttp.WithMeterProvider(noop.NewMeterProvider())) // We don't want to use the instrumentation from otelhttp

	instrumenter := metrics.NewHttpClientInstrumenter(conf.meterProvider, conf.requestDurationBuckets...)

	return resty.New().
		SetTransport(transport).
		OnBeforeRequest(func(c *resty.Client, r *resty.Request) error {
			path := r.URL
			if u, err := url.Parse(path); err == nil {
				path = u.Path
			}
			ctx := context.WithValue(r.Context(), internal.ContextKeyRestyTemplatedPath, path)
			r.SetContext(ctx)
			return conf.onBeforeRequest(c, r)
		}).
		OnAfterResponse(func(c *resty.Client, r *resty.Response) error {

			// Note: This is not full proof. If a Resty request is created without using templates,
			// it will instrument the full path which may lead to unbounded cardinality.
			path, _ := r.Request.Context().Value(internal.ContextKeyRestyTemplatedPath).(string)

			host := r.Request.RawRequest.Host
			method := r.Request.RawRequest.Method
			status := r.StatusCode()

			instrumenter.RecordRequest(r.Request.Context(), method, path, host, status, r.Time())

			// ------------------------------------------------------------------------------------
			// Record timings from httptrace if they are available
			//
			// We can't know within OnAfterResponse if the request used httptrace via Resty using
			// EnableTrace(). We make an assumption based on the data available in TraceInfo from
			// the resty.Request.
			// ------------------------------------------------------------------------------------

			traceInfo := r.Request.TraceInfo()
			traced := traceInfo.TotalTime > 0

			if traced {
				if traceInfo.DNSLookup > 0 {
					instrumenter.RecordDNSLookup(r.Request.Context(), traceInfo.DNSLookup)
				}

				if traceInfo.TLSHandshake > 0 {
					instrumenter.RecordTLSHandshake(r.Request.Context(), host, traceInfo.TLSHandshake)
				}

				if traceInfo.ConnTime > 0 {
					instrumenter.RecordConnTime(r.Request.Context(), host, traceInfo.ConnTime)
				}

				instrumenter.RecordConnIdle(r.Request.Context(), host, traceInfo.IsConnWasIdle)
				instrumenter.RecordConnReuse(r.Request.Context(), host, traceInfo.IsConnReused)
			}

			return conf.onAfterResponse(c, r)
		}).
		OnError(func(r *resty.Request, err error) {
			var respErr *resty.ResponseError
			if errors.As(err, &respErr) && respErr.Response != nil && respErr.Response.StatusCode() > 0 {
				// These metrics are already being captured in OnAfterResponse
				return
			}
			// Otherwise, we are dealing with transport level errors such as dns resolution, connection
			// timeouts, etc.
			instrumenter.RecordClientError(r.Context(), r.RawRequest.Host)

			conf.onClientError(r, err)
		})
}

type httpClientInstrumenter interface {
	RecordRequest(ctx context.Context, method, path, addr string, status int, latency time.Duration)
	RecordClientError(ctx context.Context, addr string)
	RecordDNSLookup(ctx context.Context, latency time.Duration)
	RecordTLSHandshake(ctx context.Context, addr string, latency time.Duration)
	RecordConnTime(ctx context.Context, addr string, latency time.Duration)
	RecordConnReuse(ctx context.Context, addr string, reused bool)
	RecordConnIdle(ctx context.Context, addr string, idle bool)
}
