package middleware

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"github.com/jkratz55/yuna/internal"
)

func InstrumentHandler(meterProvider metric.MeterProvider, buckets []float64) func(next http.Handler) http.Handler {

	// Not providing a MeterProvider is technically a programming error, but rather than panic, the
	// default global MeterProvider is used.
	if meterProvider == nil {
		meterProvider = otel.GetMeterProvider()
	}

	meter := meterProvider.Meter(internal.Scope, metric.WithInstrumentationVersion(internal.Version))
	requestLatency, err := meter.Float64Histogram("http.server.request.duration",
		metric.WithDescription("Duration in seconds for the server to process a request"),
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(buckets...))
	if err != nil {
		panic(err)
	}

	inFlightRequests, err := meter.Int64UpDownCounter("http.server.requests.in_flight",
		metric.WithDescription("Number of in-flight requests"))
	if err != nil {
		panic(err)
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			inFlightRequests.Add(r.Context(), 1)
			defer inFlightRequests.Add(r.Context(), -1)

			rw := newResponseWriter(w)

			start := time.Now()
			next.ServeHTTP(rw, r)
			dur := time.Since(start)

			// Because using r.URL.Path can lead to unbounded cardinality, if we cannot get the pattern
			// from Chi's context, we instrument the path as "undefined". Since YUNA uses Chi for its
			// routing, this should never happen, but this is a safeguard.
			path := "undefined"
			chiCtx := chi.RouteContext(r.Context())
			if chiCtx != nil && chiCtx.RoutePattern() != "" {
				path = chiCtx.RoutePattern()
			}

			requestLatency.Record(r.Context(), dur.Seconds(),
				metric.WithAttributes(
					attribute.Int("http.response.status_code", rw.statusCode),
					attribute.String("http.request.method", r.Method),
					attribute.String("http.route", path),
				))
		})
	}
}
