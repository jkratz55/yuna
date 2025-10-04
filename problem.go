package yuna

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"

	"go.opentelemetry.io/otel/trace"

	"github.com/jkratz55/yuna/log"
)

var (
	reservedKeys = map[string]struct{}{
		"type":       {},
		"title":      {},
		"detail":     {},
		"status":     {},
		"instance":   {},
		"violations": {},
	}
)

// ProblemDetails is a representation of a problem details object as defined by RFC 7807.
type ProblemDetails struct {
	Type       string
	Title      string
	Detail     string
	Instance   string
	StatusCode int
	Extensions map[string]interface{}

	headers http.Header
	error
}

func Problem(title string, statusCode int) *ProblemDetails {
	return &ProblemDetails{
		Type:       "about:blank",
		Title:      title,
		Detail:     "",
		Instance:   "",
		StatusCode: statusCode,
		Extensions: map[string]interface{}{},
		error:      nil,
	}
}

func (p *ProblemDetails) SetType(t string) *ProblemDetails {
	p.Type = t
	return p
}

func (p *ProblemDetails) SetDetail(d string) *ProblemDetails {
	p.Detail = d
	return p
}

func (p *ProblemDetails) SetInstance(i string) *ProblemDetails {
	p.Instance = i
	return p
}

func (p *ProblemDetails) AddExtension(key string, value interface{}) *ProblemDetails {
	if p.Extensions == nil {
		p.Extensions = make(map[string]interface{})
	}
	p.Extensions[key] = value
	return p
}

func (p *ProblemDetails) SetError(err error) *ProblemDetails {
	p.error = errors.Join(p.error, err)
	return p
}

func (p *ProblemDetails) MarshalJSON() ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteByte('{')

	writeKV := func(key string, val any, first *bool) error {
		if !*first {
			buf.WriteByte(',')
		}
		kb, err := json.Marshal(key)
		if err != nil {
			return err
		}
		vb, err := json.Marshal(val)
		if err != nil {
			return err
		}
		buf.Write(kb)
		buf.WriteByte(':')
		buf.Write(vb)
		*first = false
		return nil
	}

	first := true

	// Core fields in a fixed order
	if err := writeKV("type", p.Type, &first); err != nil {
		return nil, err
	}
	if err := writeKV("title", p.Title, &first); err != nil {
		return nil, err
	}
	if err := writeKV("detail", p.Detail, &first); err != nil {
		return nil, err
	}
	if err := writeKV("instance", p.Instance, &first); err != nil {
		return nil, err
	}
	if err := writeKV("status", p.StatusCode, &first); err != nil {
		return nil, err
	}

	// Extensions appended in sorted key order for determinism
	if len(p.Extensions) > 0 {
		keys := make([]string, 0, len(p.Extensions))
		for k := range p.Extensions {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			if _, ok := reservedKeys[k]; ok {
				continue
			}
			if err := writeKV(k, p.Extensions[k], &first); err != nil {
				return nil, err
			}
		}
	}

	buf.WriteByte('}')
	return buf.Bytes(), nil
}

func (p *ProblemDetails) Error() string {
	if p.error != nil {
		return p.error.Error()
	}
	return fmt.Sprintf("problem: %s: %s", p.Title, p.Detail)
}

func (p *ProblemDetails) Respond(w http.ResponseWriter, r *http.Request) error {
	p.ServeHTTP(w, r)
	return nil
}

func (p *ProblemDetails) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	// Default to 500 Internal Server Error if no status code is set.
	if p.StatusCode == 0 {
		p.StatusCode = http.StatusInternalServerError
	}

	if strings.TrimSpace(p.Instance) == "" {
		p.Instance = r.URL.Path
	}
	if strings.TrimSpace(p.Type) == "" {
		p.Type = "about:blank"
	}
	if p.Extensions == nil {
		p.Extensions = make(map[string]interface{})
	}

	if r.Header.Get("X-Request-ID") != "" {
		p.Extensions["requestId"] = r.Header.Get("X-Request-ID")
	}

	if r.Header.Get("X-Correlation-ID") != "" {
		p.Extensions["correlationId"] = r.Header.Get("X-Correlation-ID")
	}

	if spanCtx := trace.SpanContextFromContext(r.Context()); spanCtx.TraceID().IsValid() {
		p.Extensions["traceId"] = spanCtx.TraceID().String()
		p.Extensions["sampled"] = spanCtx.IsSampled()
	}

	for k, v := range p.headers {
		for _, h := range v {
			w.Header().Add(k, h)
		}
	}

	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(p.StatusCode)
	if err := json.NewEncoder(w).Encode(p); err != nil {
		log.LoggerFromCtx(r.Context()).Error("Error writing problem/error response to client",
			log.Error(err))
	}
}

func BadRequest(violations Violations) *ProblemDetails {
	detail := ""
	if violations == nil || len(violations) == 0 {
		detail = "Request cannot be processed because the request was not understood by the server or malformed."
	} else {
		detail = "Request validation failed. See 'violations' for details."
	}

	return &ProblemDetails{
		Type:       "about:blank",
		Title:      "Bad Request",
		Detail:     detail,
		StatusCode: http.StatusBadRequest,
		Extensions: map[string]interface{}{
			"violations": violations,
		},
	}
}

func Unauthorized() *ProblemDetails {
	return &ProblemDetails{
		Type:       "about:blank",
		Title:      "Unauthorized",
		Detail:     "Authorization is required to perform this action.",
		StatusCode: http.StatusUnauthorized,
		Extensions: make(map[string]interface{}),
	}
}

func Forbidden() *ProblemDetails {
	return &ProblemDetails{
		Type:       "about:blank",
		Title:      "Forbidden",
		Detail:     "You don't have the required role/permissions to perform this action.",
		StatusCode: http.StatusForbidden,
		Extensions: make(map[string]interface{}),
	}
}

func NotFound() *ProblemDetails {
	return &ProblemDetails{
		Type:       "about:blank",
		Title:      "Not Found",
		Detail:     "The requested resource could not be found.",
		StatusCode: http.StatusNotFound,
		Extensions: make(map[string]interface{}),
	}
}

func MethodNotAllowed(allowedMethods ...string) *ProblemDetails {
	return &ProblemDetails{
		Type:       "about:blank",
		Title:      "Method Not Allowed",
		Detail:     "The requested method is not allowed on this resource.",
		StatusCode: http.StatusMethodNotAllowed,
		Extensions: make(map[string]interface{}),
		headers: http.Header{
			"Allow": allowedMethods,
		},
	}
}

func NotAcceptable() *ProblemDetails {
	return &ProblemDetails{
		Type:       "about:blank",
		Title:      "Not Acceptable",
		Detail:     "Server cannot produce a response matching the acceptable content types.",
		StatusCode: http.StatusNotAcceptable,
	}
}

func Conflict() *ProblemDetails {
	return &ProblemDetails{
		Type:       "about:blank",
		Title:      "Conflict",
		Detail:     "The request could not be completed due to a conflict.",
		StatusCode: http.StatusConflict,
		Extensions: make(map[string]interface{}),
	}
}

func Gone() *ProblemDetails {
	return &ProblemDetails{
		Type:       "about:blank",
		Title:      "Gone",
		Detail:     "The requested resource is no longer available.",
		StatusCode: http.StatusGone,
		Extensions: make(map[string]interface{}),
	}
}

func UnsupportedMediaType() *ProblemDetails {
	return &ProblemDetails{
		Type:       "about:blank",
		Title:      "Unsupported Media Type",
		Detail:     "The request entity has a media type which the server or resource does not support.",
		StatusCode: http.StatusUnsupportedMediaType,
		Extensions: make(map[string]interface{}),
	}
}

func UnprocessableEntity(violations Violations) *ProblemDetails {
	detail := ""
	if violations == nil || len(violations) == 0 {
		detail = "The request entity could not be processed."
	} else {
		detail = "The request entity could not be processed due to validation errors. See 'violations' for details."
	}
	prob := &ProblemDetails{
		Type:       "about:blank",
		Title:      "Unprocessable Entity",
		Detail:     detail,
		StatusCode: http.StatusUnprocessableEntity,
		Extensions: make(map[string]interface{}),
	}
	if len(violations) > 0 {
		prob.Extensions["violations"] = violations
	}
	return prob
}

func TooManyRequests() *ProblemDetails {
	return &ProblemDetails{
		Type:       "about:blank",
		Title:      "Too Many Requests",
		Detail:     "The request was rate-limited. Try again later.",
		StatusCode: http.StatusTooManyRequests,
		Extensions: make(map[string]interface{}),
	}
}

func InternalServerError(errs ...error) *ProblemDetails {
	pd := &ProblemDetails{
		Type:       "about:blank",
		Title:      "Internal Server Error",
		Detail:     "Server encountered an internal error processing the request. Please try again later.",
		StatusCode: http.StatusInternalServerError,
		Extensions: make(map[string]interface{}),
	}
	if len(errs) > 0 {
		err := errors.Join(errs...)
		pd.error = err
	}
	return pd
}

func BadGateway() *ProblemDetails {
	return &ProblemDetails{
		Type:       "about:blank",
		Title:      "Bad Gateway",
		Detail:     "The server received an invalid response from an upstream server.",
		StatusCode: http.StatusBadGateway,
		Extensions: make(map[string]interface{}),
	}
}

func ServiceUnavailable() *ProblemDetails {
	return &ProblemDetails{
		Type:       "about:blank",
		Title:      "Service Unavailable",
		Detail:     "The server is currently unable to handle the request due to a temporary overloading or maintenance of the server. Please try again later.",
		StatusCode: http.StatusServiceUnavailable,
		Extensions: make(map[string]interface{}),
	}
}

func GatewayTimeout() *ProblemDetails {
	return &ProblemDetails{
		Type:       "about:blank",
		Title:      "Gateway Timeout",
		Detail:     "The server timed out while proxying the request to the upstream server. Please try again later.",
		StatusCode: http.StatusGatewayTimeout,
	}
}

// Violations represent validation errors / constraint violations detected during request validation.
//
// The key is the field name, and the value is a slice of error/constraint violations. Violations are
// intended to be used with ProblemDetails to describe to the client why the request is invalid and
// the server refused to process it.
type Violations map[string][]string

// Add adds a violation message for the given field.
//
// If the field already has a violation message, the new message is appended to the existing list.
func (v *Violations) Add(field string, errs ...string) {
	if *v == nil {
		*v = make(Violations)
	}
	m := *v
	violation, ok := m[field]
	if !ok {
		violation = make([]string, 0, len(errs))
	}
	m[field] = append(violation, errs...)
}
