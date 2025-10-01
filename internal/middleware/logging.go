package middleware

import (
	"context"
	"net/http"

	"go.opentelemetry.io/otel/trace"

	"github.com/jkratz55/yuna/internal"
	"github.com/jkratz55/yuna/log"
)

func RequestLogger(logger *log.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if requestID := r.Header.Get("X-Request-ID"); requestID != "" {
				logger = logger.With(log.String("request_id", requestID))
			}
			if correlationID := r.Header.Get("X-Correlation-ID"); correlationID != "" {
				logger = logger.With(log.String("correlation_id", correlationID))
			}
			if spanCtx := trace.SpanContextFromContext(r.Context()); spanCtx.TraceID().IsValid() {
				logger = logger.With(log.String("trace_id", spanCtx.TraceID().String()))
			}

			requestInfo := map[string]string{}
			requestInfo["method"] = r.Method
			requestInfo["path"] = r.URL.Path
			requestInfo["uri"] = r.URL.RequestURI()

			if r.Header.Get("Origin") != "" {
				requestInfo["origin"] = r.Header.Get("Origin")
			}
			if r.Header.Get("User-Agent") != "" {
				requestInfo["user_agent"] = r.Header.Get("User-Agent")
			}

			logger = logger.With(log.Any("http_request", requestInfo))

			ctx := context.WithValue(r.Context(), internal.ContextKeyLogger, logger)
			r = r.WithContext(ctx)

			next.ServeHTTP(w, r)
		})
	}
}
