package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.37.0"

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

	shutdownFn, err := initOpenTelemtry()
	if err != nil {
		logger.Panic(fmt.Sprintf("Failed to initialize OpenTelemetry: %s", err))
		return
	}
	defer shutdownFn()

	// --------------------------------------------------------------------------------------------
	// The meat of the example
	// --------------------------------------------------------------------------------------------

	// Setup silly HttpAuthenticator implementation that always assumes the user is an admin is any
	// API token is present in the request header.
	authenticator := &TokenAuthenticator{}

	// Create a new application with metrics exposed via Prometheus, PPROF, and health checks
	app := yuna.New(
		yuna.WithMetrics(),
		yuna.WithPPROF(),
		yuna.WithHealthChecks(),
		yuna.WithAuthentication(authenticator))

	app.RegisterHealthCheck(yuna.ComponentRegistration{
		Name:     "Silly",
		Critical: true,
		Checker: yuna.HealthCheckerFunc(func(ctx context.Context) yuna.HealthStatus {
			return yuna.StatusUp
		}),
		Tags:    []string{"silly"},
		Timeout: time.Second * 1,
	})

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
		}, yuna.Consumes(yuna.MIMEApplicationJSON))

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

		r.Get("/silly", sillyHandler, yuna.Authenticated(), sillyMiddleware(), sillyMiddleware2())

		oh := &OrderHandler{}
		r.Get("/order", oh.getOrder, sillyMiddleware(), sillyMiddleware2())

		r.Get("/responderfunc", func(r *yuna.Request) yuna.Responder {
			return yuna.ResponderFunc(func(w http.ResponseWriter, r *http.Request) error {
				w.WriteHeader(http.StatusOK)
				return nil
			})
		})
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

// ------------------------------------------------------------------------------------------------
// Initialize and configure OpenTelemetry
// ------------------------------------------------------------------------------------------------

func initOpenTelemtry() (func() error, error) {

	// Setup OpenTelemetry trace exporter
	traceExporter, err := otlptracehttp.New(context.Background())
	if err != nil {
		return nil, err
	}

	// Configure OpenTelemetry resource and TracerProvider
	otelResource := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceNameKey.String("cacheotel-example"),
		semconv.ServiceVersionKey.String("1.0.0"))
	traceProvider := trace.NewTracerProvider(
		trace.WithBatcher(traceExporter),
		trace.WithResource(otelResource))

	// Set the TraceProvider and TextMapPropagator globally
	otel.SetTracerProvider(traceProvider)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{}))

	// Setup OpenTelemetry metric exporter
	exporter, err := prometheus.New()
	if err != nil {
		return nil, err
	}

	provider := metric.NewMeterProvider(metric.WithReader(exporter))
	otel.SetMeterProvider(provider)

	shutdownFn := func() error {
		var err error
		errors.Join(err, traceProvider.Shutdown(context.Background()))
		errors.Join(err, provider.Shutdown(context.Background()))
		return err
	}

	return shutdownFn, nil
}

// ------------------------------------------------------------------------------------------------
// Implementing authentication
// ------------------------------------------------------------------------------------------------

type TokenAuthenticator struct{}

func (t *TokenAuthenticator) Authenticate(r *http.Request) (yuna.Principal, error) {

	token := r.Header.Get("X-API-Token")
	if strings.TrimSpace(token) == "" {
		return &UserPrincipal{
			name:      "",
			id:        "",
			anonymous: true,
		}, nil
	}

	// Silly example, if the token is present, always assume they are logged in
	return &UserPrincipal{
		name:      "test-admin",
		id:        "test-admin",
		anonymous: false,
	}, nil
}

type UserPrincipal struct {
	name      string
	id        string
	anonymous bool
}

func (u *UserPrincipal) Name() string {
	return u.name
}

func (u *UserPrincipal) SubjectID() string {
	return u.id
}

func (u *UserPrincipal) Anonymous() bool {
	return u.anonymous
}

func (u *UserPrincipal) HasRole(role string) bool {
	return false
}

func (u *UserPrincipal) Attribute(key string) (any, bool) {
	return nil, false
}
