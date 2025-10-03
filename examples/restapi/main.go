package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"

	"github.com/jkratz55/yuna"
	"github.com/jkratz55/yuna/log"
)

func main() {

	// --------------------------------------------------------------------------------------------
	// Setup logging and telemetry
	// --------------------------------------------------------------------------------------------

	logger := log.GetLogger()
	logger.Info("Initializing application")

	otel.SetErrorHandler(otel.ErrorHandlerFunc(func(err error) {
		logger := log.New(log.WithLevel(log.LevelError)).With(slog.String("logger", "opentelemetry"))
		logger.Error(fmt.Sprintf("%s", err))
	}))

	// Setup OpenTelemetry trace exporter
	traceExporter, err := otlptracehttp.New(context.Background())
	if err != nil {
		logger.Error("error creating trace exporter", slog.String("err", err.Error()))
		panic(err)
	}

	// Configure OpenTelemetry resource and TracerProvider
	otelResource := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceNameKey.String("cacheotel-example"),
		semconv.ServiceVersionKey.String("1.0.0"))
	traceProvider := trace.NewTracerProvider(
		trace.WithBatcher(traceExporter),
		trace.WithResource(otelResource))
	defer func() {
		err := traceProvider.Shutdown(context.Background())
		if err != nil {
			logger.Error("error shutting down trace provider", slog.String("err", err.Error()))
		}
	}()

	// Set the TraceProvider and TextMapPropagator globally
	otel.SetTracerProvider(traceProvider)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{}))

	// Setup OpenTelemetry metric exporter
	exporter, err := prometheus.New()
	if err != nil {
		logger.Error("error creating prometheus exporter", slog.String("err", err.Error()))
		panic(err)
	}
	provider := metric.NewMeterProvider(metric.WithReader(exporter))
	otel.SetMeterProvider(provider)
	defer provider.Shutdown(context.Background())

	// --------------------------------------------------------------------------------------------
	// The meat of the example
	// --------------------------------------------------------------------------------------------

	// Create a new application with metrics exposed via Prometheus, PPROF, and health checks
	app := yuna.New(
		yuna.WithMetrics(),
		yuna.WithPPROF(),
		yuna.WithHealthChecks())

	app.Get("/hello", func(r *yuna.Request) yuna.Responder {
		return yuna.Ok(map[string]string{"hello": "world"})
	})

	app.Route("/api", func(r yuna.Router) {

		r.Post("/event", func(r *yuna.Request) yuna.Responder {

			type createEventRequest struct {
				Name        string   `json:"name"`
				Description string   `json:"description"`
				Tags        []string `json:"tags"`
			}

			// Using Decode will parse the request body into the struct based on the Content-Type. It
			// supports JSON, XML, and msgpack.
			var req createEventRequest
			err := r.Decode(&req)
			if err != nil {
				return yuna.BadRequest(nil)
			}

			// todo: could do validation of request here as well

			// todo: do business logic here

			// Return HTTP 201 Created with the location of the newly created resource
			return yuna.Created(fmt.Sprintf("/api/event/%s", uuid.New().String()))
		})

		r.Get("/event", func(r *yuna.Request) yuna.Responder {

			type requestParams struct {
				Envs []string `form:"env"`
				Tags []string `form:"tag"`
			}

			var params requestParams
			err := r.Bind(&params)
			if err != nil {
				return yuna.BadRequest(nil)
			}

			// Silly print to prove bind works
			fmt.Println(params)

			// todo: could do validation of request here as well

			// todo: do business logic here

			return yuna.NoContent()
		})

		r.Get("/event-manual", func(r *yuna.Request) yuna.Responder {

			type requestParams struct {
				Envs []string
				Tags []string
			}

			var params requestParams
			params.Tags = r.QueryParam("tag").Values()
			params.Envs = r.QueryParam("env").Values()

			// Silly print to prove fetching query params works
			fmt.Println(params)

			// todo: could do validation of request here as well

			// todo: do business logic here

			return yuna.NoContent()
		})

		r.Get("/silly", sillyHandler, sillyMiddleware(), sillyMiddleware2())

		oh := &OrderHandler{}
		r.Get("/order", oh.getOrder, sillyMiddleware(), sillyMiddleware2())
	})

	app.Start()
}

func sillyHandler(r *yuna.Request) yuna.Responder {
	return yuna.Ok(map[string]string{"message": "GET SILLY!"})
}

func sillyMiddleware() yuna.HttpMiddleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Println("silly middleware: start")
			next.ServeHTTP(w, r)
			fmt.Println("silly middleware: end")
		})
	}
}

func sillyMiddleware2() yuna.HttpMiddleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Println("silly middleware 2: start")
			next.ServeHTTP(w, r)
			fmt.Println("silly middleware 2: end")
		})
	}
}

type OrderHandler struct{}

func (h *OrderHandler) getOrder(r *yuna.Request) yuna.Responder {
	return yuna.Ok(map[string]string{"message": "GET ORDER!"})
}
