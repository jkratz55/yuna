package metrics

import (
	"context"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	semconv "go.opentelemetry.io/otel/semconv/v1.37.0"

	"github.com/jkratz55/yuna/internal"
)

type HttpClientInstrumenter struct {
	requestDurationHistogram metric.Float64Histogram
	clientErrorsCounter      metric.Int64Counter
	dnsLookupHistogram       metric.Float64Histogram
	tlsHandshakeHistogram    metric.Float64Histogram
	connTimeHistogram        metric.Float64Histogram
	connReuseCounter         metric.Int64Counter
	connIdleCounter          metric.Int64Counter
}

func NewHttpClientInstrumenter(meterProvider metric.MeterProvider, buckets ...float64) *HttpClientInstrumenter {

	if meterProvider == nil {
		meterProvider = otel.GetMeterProvider()
	}
	if len(buckets) == 0 {
		buckets = []float64{0.025, 0.050, 0.100, 0.250, 0.500, 1}
	}

	meter := meterProvider.Meter(internal.Scope, metric.WithInstrumentationVersion(internal.Version))

	requestDurationHistogram, err := meter.Float64Histogram("http.client.request.duration",
		metric.WithDescription("Duration in seconds for the client to transmit a request and receive a response from the server"),
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(buckets...))
	if err != nil {
		panic(err)
	}

	clientErrorsCounter, err := meter.Int64Counter("http.client.errors",
		metric.WithDescription("Number of errors encountered by the client"))
	if err != nil {
		panic(err)
	}

	dnsLookupHistogram, err := meter.Float64Histogram("http.client.dns.lookup.duration",
		metric.WithDescription("Duration in seconds for the client to perform a DNS lookup"),
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(0.010, 0.025, 0.050, 0.100, 0.250))
	if err != nil {
		panic(err)
	}

	tlsHandshakeHistogram, err := meter.Float64Histogram("http.client.tls.handshake.duration",
		metric.WithDescription("Duration in seconds for the client to perform a TLS handshake"),
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(0.010, 0.025, 0.050, 0.100, 0.250))
	if err != nil {
		panic(err)
	}

	connTimeHistogram, err := meter.Float64Histogram("http.client.connection.duration",
		metric.WithDescription("Duration in seconds for the client to establish a connection to the server"),
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(0.010, 0.025, 0.050, 0.100, 0.250))
	if err != nil {
		panic(err)
	}

	connReuseCounter, err := meter.Int64Counter("http.client.connection.reuse",
		metric.WithDescription("Number of times the client reused a connection to the server"))
	if err != nil {
		panic(err)
	}

	connIdleCounter, err := meter.Int64Counter("http.client.connection.idle",
		metric.WithDescription("Number of times the client closed a connection to the server"))
	if err != nil {
		panic(err)
	}

	return &HttpClientInstrumenter{
		requestDurationHistogram: requestDurationHistogram,
		clientErrorsCounter:      clientErrorsCounter,
		dnsLookupHistogram:       dnsLookupHistogram,
		tlsHandshakeHistogram:    tlsHandshakeHistogram,
		connTimeHistogram:        connTimeHistogram,
		connReuseCounter:         connReuseCounter,
		connIdleCounter:          connIdleCounter,
	}
}

func (h *HttpClientInstrumenter) RecordRequest(ctx context.Context, method, path, addr string,
	status int, latency time.Duration) {

	h.requestDurationHistogram.Record(ctx, latency.Seconds(),
		metric.WithAttributes(
			semconv.HTTPRequestMethodKey.String(method),
			semconv.ServerAddressKey.String(addr),
			semconv.HTTPResponseStatusCodeKey.Int(status),
			semconv.HTTPRouteKey.String(path)))
}

func (h *HttpClientInstrumenter) RecordClientError(ctx context.Context, addr string) {
	h.clientErrorsCounter.Add(ctx, 1, metric.WithAttributes(
		semconv.ServerAddressKey.String(addr)))
}

func (h *HttpClientInstrumenter) RecordDNSLookup(ctx context.Context, latency time.Duration) {
	h.dnsLookupHistogram.Record(ctx, latency.Seconds())
}

func (h *HttpClientInstrumenter) RecordTLSHandshake(ctx context.Context, addr string, latency time.Duration) {
	h.tlsHandshakeHistogram.Record(ctx, latency.Seconds(), metric.WithAttributes(
		semconv.ServerAddressKey.String(addr)))
}

func (h *HttpClientInstrumenter) RecordConnTime(ctx context.Context, addr string, latency time.Duration) {
	h.connTimeHistogram.Record(ctx, latency.Seconds(), metric.WithAttributes(
		semconv.ServerAddressKey.String(addr)))
}

func (h *HttpClientInstrumenter) RecordConnReuse(ctx context.Context, addr string, reused bool) {
	h.connReuseCounter.Add(ctx, 1, metric.WithAttributes(
		semconv.ServerAddressKey.String(addr),
		attribute.Bool("reused", reused)))
}

func (h *HttpClientInstrumenter) RecordConnIdle(ctx context.Context, addr string, idle bool) {
	h.connIdleCounter.Add(ctx, 1, metric.WithAttributes(
		semconv.ServerAddressKey.String(addr),
		attribute.Bool("was.idle", idle)))
}
