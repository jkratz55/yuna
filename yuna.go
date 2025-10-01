package yuna

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"runtime/debug"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/jkratz55/yuna/internal/middleware"
	"github.com/jkratz55/yuna/log"
)

type Yuna struct {
	router        chi.Router
	server        *http.Server
	opServer      *http.Server
	config        *config
	healthHandler *healthcheckHandler
	logger        *log.Logger
	startTs       time.Time
}

func New(opts ...ServerOption) *Yuna {

	baseOpts := make([]baseOption, len(opts))
	for i, opt := range opts {
		baseOpts[i] = opt
	}
	conf := newConfig(baseOpts...)

	z := &Yuna{
		router:        chi.NewRouter(),
		config:        conf,
		healthHandler: newHealthcheckHandler(),
		logger:        conf.logger,
	}

	httpServer := &http.Server{
		Addr:              fmt.Sprintf(":%d", conf.httpPort),
		Handler:           z,
		ReadTimeout:       conf.readTimeout,
		ReadHeaderTimeout: conf.readHeaderTimeout,
		WriteTimeout:      conf.writeTimeout,
		IdleTimeout:       conf.idleTimeout,
		BaseContext:       conf.baseContext,
	}
	z.server = httpServer

	z.opServer = z.initOpsServer(conf)

	// Setup default middleware
	z.router.Use(recovery())
	z.router.Use(middleware.Trace(conf.traceProvider, z))
	z.router.Use(middleware.InstrumentHandler(conf.meterProvider, conf.requestDurationBuckets))
	z.router.Use(middleware.RequestLogger(conf.logger))

	// Setup default handlers for Chi if the route doesn't match or the method is not allowed
	z.router.NotFound(conf.notFoundHandler.ServeHTTP)
	z.router.MethodNotAllowed(conf.methodNotAllowedHandler.ServeHTTP)

	return z
}

func (z *Yuna) initOpsServer(conf *config) *http.Server {
	opMux := chi.NewMux()
	if conf.metricsEnabled {
		opMux.Mount("/metrics", promhttp.Handler())
	}
	if conf.pprofEnabled {
		opMux.Mount("/debug/pprof/", http.DefaultServeMux)
	}
	if conf.healthcheckEnabled {
		opMux.Mount(conf.healthcheckBasePath, z.healthHandler)
	}

	opMux.Get("/info", func(w http.ResponseWriter, r *http.Request) {
		buildInfo, ok := debug.ReadBuildInfo()
		if !ok {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		w.Header().Set(HeaderContentType, MIMEApplicationJSON)
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(buildInfo)
	})

	opMux.Get("/uptime", func(w http.ResponseWriter, r *http.Request) {
		type resp struct {
			Uptime string    `json:"uptime"`
			Start  time.Time `json:"start"`
		}

		res := resp{
			Uptime: time.Since(z.startTs).String(),
			Start:  z.startTs,
		}

		w.Header().Set(HeaderContentType, MIMEApplicationJSON)
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(res)
	})

	opMux.Put("/log/level", func(w http.ResponseWriter, r *http.Request) {

		type request struct {
			Level string `json:"level"`
		}

		type response struct {
			Success bool   `json:"success"`
			Error   string `json:"error,omitempty"`
		}

		var payload request
		err := json.NewDecoder(r.Body).Decode(&payload)
		if err != nil {
			w.Header().Set(HeaderContentType, MIMEApplicationJSON)
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(response{
				Success: false,
				Error:   err.Error(),
			})
			return
		}

		lvl, err := log.ParseLevel(payload.Level)
		if err != nil {
			w.Header().Set(HeaderContentType, MIMEApplicationJSON)
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(response{
				Success: false,
				Error:   err.Error(),
			})
			return
		}

		// NOTE: Only the global logger and logger set on Yuna are updated. If loggers are created
		// using [log.NewLogger] their levels cannot be updated through this endpoint.
		z.logger.SetLevel(lvl)
		log.GetLogger().SetLevel(lvl)

		w.Header().Set(HeaderContentType, MIMEApplicationJSON)
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(response{
			Success: true,
		})

		z.logger.Info(fmt.Sprintf("Log Level is now set to %s", payload.Level))
	})

	return &http.Server{
		Addr:    fmt.Sprintf(":%d", conf.operationHTTPPort),
		Handler: opMux,
	}
}

// Start begins listening and serving HTTP requests.
//
// Start blocks until the server is stopped or an error occurs. Generally, Start should be called in a
// separate goroutine. Start begins listening and serving HTTP requests on the main HTTP server,
// and operations HTTP server. If either server encounters an error Start will return an error and
// the Yuna instance is no longer usable.
func (z *Yuna) Start() error {

	z.startTs = time.Now()
	errs := make(chan error, 2)

	go func() {
		z.logger.Info(fmt.Sprintf("Starting operations HTTP server on port %d", z.config.operationHTTPPort))
		err := z.opServer.ListenAndServe()
		errs <- fmt.Errorf("operations HTTP server: %w", err)
	}()

	go func() {
		z.logger.Info(fmt.Sprintf("Starting main HTTP server on port %d", z.config.httpPort))
		err := z.server.ListenAndServe()
		errs <- fmt.Errorf("main HTTP server: %w", err)
	}()

	err := <-errs
	if !errors.Is(err, http.ErrServerClosed) {
		_ = z.opServer.Close()
		_ = z.server.Close()
		return fmt.Errorf("server stopped with error: %w", err)
	}

	err = <-errs
	if !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("server stopped with error: %w", err)
	}

	return nil
}

// StartTLS begins listening and serving HTTPS requests.
//
// StartTLS blocks until the server is stopped or an error occurs. Generally, StartTLS should be called in a
// separate goroutine. StartTLS begins listening and serving HTTPS requests on the main HTTP server,
// while the Prometheus HTTP server and optionally pprof HTTP server if enabled, use HTTP instead of HTTPS.
func (z *Yuna) StartTLS(certFile, keyFile string) error {

	z.startTs = time.Now()
	errs := make(chan error, 2)

	go func() {
		z.logger.Info(fmt.Sprintf("Starting operations HTTP server on port %d", z.config.operationHTTPPort))
		err := z.opServer.ListenAndServe()
		errs <- fmt.Errorf("operations HTTP server: %w", err)
	}()

	go func() {
		z.logger.Info(fmt.Sprintf("Starting main HTTP server on port %d", z.config.httpPort))
		err := z.server.ListenAndServeTLS(certFile, keyFile)
		errs <- fmt.Errorf("main HTTP server: %w", err)
	}()

	err := <-errs
	if !errors.Is(err, http.ErrServerClosed) {
		_ = z.opServer.Close()
		_ = z.server.Close()
		return fmt.Errorf("server stopped with error: %w", err)
	}

	err = <-errs
	if !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("server stopped with error: %w", err)
	}

	return nil
}

// Shutdown gracefully shuts down the server without interrupting any active connections and inflight
// requests. Shutdown accepts a time.Duration which represents the maximum duration to wait for in-flight
// requests to complete. After the timeout passes, Shutdown will forcefully close all connections impacting
// any in-flight requests.
func (z *Yuna) Shutdown(terminationGraceDuration time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), terminationGraceDuration)
	defer cancel()

	go func() {
		_ = z.opServer.Shutdown(ctx)
	}()

	return z.server.Shutdown(ctx)
}

// Close immediately closes all active listeners and all connections.
//
// For a graceful shutdown, use Shutdown instead.
func (z *Yuna) Close() error {
	// Errors from pprof and prometheus servers are ignored by design as it isn't important if those
	// return an error during application shutdown.
	_ = z.opServer.Close()
	return z.server.Close()
}

// RegisterOnShutdown registers a function to be called when the server is shutting down. This can be
// used to gracefully shutdown connections that have undergone ALPN protocol upgrade or that have been
// hijacked.
func (z *Yuna) RegisterOnShutdown(f func()) {
	z.server.RegisterOnShutdown(f)
}

// ServeHTTP serves an HTTP request.
//
// This implements the http.Handler interface.
func (z *Yuna) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	z.router.ServeHTTP(w, r)
}

func (z *Yuna) Use(middleware ...func(http.Handler) http.Handler) {
	z.router.Use(middleware...)
}

func (z *Yuna) With(middleware ...func(http.Handler) http.Handler) Router {
	router := z.router.With(middleware...)
	return &Mux{r: router}
}

func (z *Yuna) Get(pattern string, fn HandlerFunc) {
	z.router.Get(pattern, wrapFn(fn))
}

func (z *Yuna) Post(pattern string, fn HandlerFunc) {
	z.router.Post(pattern, wrapFn(fn))
}

func (z *Yuna) Put(pattern string, fn HandlerFunc) {
	z.router.Put(pattern, wrapFn(fn))
}

func (z *Yuna) Delete(pattern string, fn HandlerFunc) {
	z.router.Delete(pattern, wrapFn(fn))
}

func (z *Yuna) Patch(pattern string, fn HandlerFunc) {
	z.router.Patch(pattern, wrapFn(fn))
}

func (z *Yuna) Options(pattern string, fn HandlerFunc) {
	z.router.Options(pattern, wrapFn(fn))
}

func (z *Yuna) Head(pattern string, fn HandlerFunc) {
	z.router.Head(pattern, wrapFn(fn))
}

func (z *Yuna) Connect(pattern string, fn HandlerFunc) {
	z.router.Connect(pattern, wrapFn(fn))
}

func (z *Yuna) Trace(pattern string, fn HandlerFunc) {
	z.router.Trace(pattern, wrapFn(fn))
}

func (z *Yuna) Method(method, pattern string, handler Handler) {
	z.router.Method(method, pattern, wrap(handler))
}

func (z *Yuna) Mount(pattern string, h http.Handler) {
	z.router.Mount(pattern, h)
}

func (z *Yuna) Route(pattern string, fn func(r Router)) {
	z.router.Route(pattern, func(r chi.Router) {
		fn(&Mux{r: r})
	})
}

func (z *Yuna) Group(fn func(r Router)) {
	z.router.Group(func(r chi.Router) {
		fn(&Mux{r: r})
	})
}

func (z *Yuna) Routes() []chi.Route {
	return z.router.Routes()
}

func (z *Yuna) Middlewares() chi.Middlewares {
	return z.router.Middlewares()
}

func (z *Yuna) Match(rctx *chi.Context, method, path string) bool {
	return z.router.Match(rctx, method, path)
}

func (z *Yuna) Find(rctx *chi.Context, method, path string) string {
	return z.router.Find(rctx, method, path)
}

func (z *Yuna) RegisterHealthCheck(component ComponentRegistration) {
	if !z.config.healthcheckEnabled {
		z.logger.Warn("Health checks are not enabled. This operation will have no impact.")
	}
	if component.Timeout == 0 {
		z.logger.Warn(fmt.Sprintf("Health check %s has no timeout. Setting to 1 second.", component.Name))
		component.Timeout = time.Second * 1
	}
	z.healthHandler.register(component)
}

func notFound(_ *Request) Responder {
	return NotFound()
}

func methodNotAllowed(_ *Request) Responder {
	return MethodNotAllowed()
}
