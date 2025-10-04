package yuna

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
)

type HealthStatus string

const (
	StatusUp       HealthStatus = "UP"
	StatusDegraded HealthStatus = "DEGRADED"
	StatusDown     HealthStatus = "DOWN"
)

var healthStatusToCode = map[HealthStatus]int{
	StatusUp:       http.StatusOK,
	StatusDegraded: http.StatusOK,
	StatusDown:     http.StatusServiceUnavailable,
}

func (s HealthStatus) String() string {
	return string(s)
}

func (s HealthStatus) StatusCode() int {
	code, ok := healthStatusToCode[s]
	if !ok {
		// In theory this should never happen, but just in case...
		return http.StatusServiceUnavailable
	}
	return code
}

type HealthChecker interface {
	Check(ctx context.Context) HealthStatus
}

type HealthCheckerFunc func(ctx context.Context) HealthStatus

func (f HealthCheckerFunc) Check(ctx context.Context) HealthStatus {
	return f(ctx)
}

type ComponentRegistration struct {
	Name     string
	Critical bool
	Checker  HealthChecker
	Tags     []string
	Timeout  time.Duration
}

type HealthResponse struct {
	Status     HealthStatus `json:"status"`
	Components []Component  `json:"components"`
	Timestamp  time.Time    `json:"timestamp"`
}

type Component struct {
	Name   string       `json:"name"`
	Status HealthStatus `json:"status"`
	Tags   []string     `json:"tags"`
}

type healthcheckHandler struct {
	components []ComponentRegistration
	router     chi.Router
}

func newHealthcheckHandler() *healthcheckHandler {
	handler := &healthcheckHandler{
		router:     chi.NewRouter(),
		components: make([]ComponentRegistration, 0),
	}

	handler.router.Get("/live", handler.live)
	handler.router.Get("/ready", handler.ready)
	return handler
}

func (h *healthcheckHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.router.ServeHTTP(w, r)
}

func (h *healthcheckHandler) register(c ComponentRegistration) {
	h.components = append(h.components, c)
}

func (h *healthcheckHandler) live(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"UP"}`))
}

func (h *healthcheckHandler) ready(w http.ResponseWriter, r *http.Request) {
	resp := readyStatus(r.Context(), h.components)

	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")
	w.Header().Set("X-Health-Status", resp.Status.String())
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.Status.StatusCode())
	_ = json.NewEncoder(w).Encode(resp)
}

func readyStatus(ctx context.Context, components []ComponentRegistration) HealthResponse {

	type result struct {
		name     string
		critical bool
		status   HealthStatus
		tags     []string
	}

	var wg sync.WaitGroup
	results := make(chan result, len(components))

	// All health checks are run concurrently so that a single slow check does not block and possibly
	// cause the health probe to timeout.
	for _, c := range components {
		wg.Add(1)
		go func(comp ComponentRegistration) {
			defer wg.Done()

			ctx, cancel := context.WithTimeout(ctx, comp.Timeout)
			defer cancel()

			status := comp.Checker.Check(ctx)
			results <- result{
				name:     comp.Name,
				critical: comp.Critical,
				status:   status,
				tags:     comp.Tags,
			}
		}(c)
	}

	wg.Wait()
	close(results)

	response := HealthResponse{
		Status:     StatusUp,
		Components: make([]Component, 0, len(components)),
		Timestamp:  time.Now(),
	}
	for res := range results {
		response.Components = append(response.Components, Component{
			Name:   res.name,
			Status: res.status,
			Tags:   res.tags,
		})
		switch res.status {
		case StatusUp:
			// Nothing to do, the overall status starts as UP and once the overall status is downgraded
			// it doesn't get upgraded.
		case StatusDegraded:
			// Once the overall status is downgraded to DOWN it should not be overwritten
			if response.Status != StatusDown {
				response.Status = StatusDegraded
			}
		case StatusDown:
			// If the component is marked as critical and it's down, the overall status is considered
			// down.
			if res.critical {
				response.Status = StatusDown
			}

			// If the component is not marked as critical and is down, the overall status is downgraded.
			// However, if the overall status is already down, then it remains down.
			if !res.critical && response.Status != StatusDown {
				response.Status = StatusDegraded
			}
		}
	}

	return response
}
