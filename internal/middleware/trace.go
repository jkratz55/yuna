package middleware

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/trace"
)

func Trace(tp trace.TracerProvider, routes chi.Routes, opts ...otelhttp.Option) func(next http.Handler) http.Handler {

	// Not providing a TracerProvider is technically a programming error, but rather than panic, the
	// default global TracerProvider is used.
	if tp == nil {
		tp = otel.GetTracerProvider()
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fn := func(w http.ResponseWriter, r *http.Request) {

				// If an X-Request-ID header is not set, a random one is generated and set. To make
				// it convenient for the caller and debugging, we always return the X-Request-ID
				// and the X-Correlation-ID if it is available.
				if r.Header.Get("X-Request-ID") == "" {
					r.Header.Set("X-Request-ID", uuid.New().String())
				}
				w.Header().Set("X-Request-ID", r.Header.Get("X-Request-ID"))

				if r.Header.Get("X-Correlation-ID") != "" {
					w.Header().Set("X-Correlation-ID", r.Header.Get("X-Correlation-ID"))
				}

				// If there is a valid active span set the X-Trace-Id and X-Sampled headers.
				spanCtx := trace.SpanContextFromContext(r.Context())
				if spanCtx.HasTraceID() {
					w.Header().Set("X-Trace-Id", spanCtx.TraceID().String())
				}
				if spanCtx.IsSampled() {
					w.Header().Set("X-Sampled", "1")
				} else {
					w.Header().Set("X-Sampled", "0")
				}

				next.ServeHTTP(w, r)
			}

			// A custom formatter is used so that this middleware can be used at the top-level while
			// still naming the span based on the matched route/pattern.
			formatter := func(operation string, r *http.Request) string {
				// Attempts to match the request to the route without executing the handlers.
				chiCtx := chi.NewRouteContext()
				if routes != nil && routes.Match(chiCtx, r.Method, r.URL.Path) {
					pat := chiCtx.RoutePattern()
					if pat != "" {
						return r.Method + " " + pat
					}
				}
				return r.Method + " " + r.URL.Path
			}

			// A noop MeterProvider is intentionally used here so that it does not interfere with
			// yuna's own middleware to capture metrics. The otelhttp implementation does not capture
			// metrics at the path level, which makes it difficult to understand the performance of
			// individual endpoints.
			opts = append(opts,
				otelhttp.WithTracerProvider(tp),
				otelhttp.WithSpanNameFormatter(formatter),
				otelhttp.WithMeterProvider(noop.NewMeterProvider()),
			)

			otelHandler := otelhttp.NewHandler(http.HandlerFunc(fn), "", opts...)
			otelHandler.ServeHTTP(w, r)
		})
	}
}
