package yuna

import (
	"context"
	"net/http"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"

	"github.com/jkratz55/yuna/internal"
)

// An HttpAuthenticator authenticates the user/client from the request.
//
// Implementations of HttpAuthenticator are expected to extract the credentials from the [http.Request],
// validate them, and return a Principal representing the authenticated user/client.
//
// Implementations of HttpAuthenticator should only return a non-nil error value if an error occurs
// while authenticating, not if the request could not be authenticated. If the credentials are missing
// or invalid, a Principal should be returned where Principal.Authenticated() returns false. If a
// nil error is returned, Principal must be non-nil.
//
// Implementations of HttpAuthenticator must be safe for concurrent use by multiple goroutines.
type HttpAuthenticator interface {
	Authenticate(r *http.Request) (Principal, error)
}

// A Principal represents the authentication state of a user/client.
type Principal interface {
	// Name returns the name of the authenticated user/client.
	Name() string

	// SubjectID returns the stable identifier for the authenticated user/client.
	SubjectID() string

	// Anonymous returns true if the user/client is anonymous / unauthenticated.
	Anonymous() bool

	// HasRole returns true if the user/client has the specified role.
	HasRole(role string) bool

	// Attribute returns the value of an attribute if present.
	Attribute(key string) (any, bool)
}

// WithPrincipal adds a Principal to the context.
func WithPrincipal(ctx context.Context, p Principal) context.Context {
	return context.WithValue(ctx, internal.ContextKeyPrincipal, p)
}

// PrincipalFromCtx returns the Principal from the context, if present.
func PrincipalFromCtx(ctx context.Context) (Principal, bool) {
	p, ok := ctx.Value(internal.ContextKeyPrincipal).(Principal)
	return p, ok
}

func Authenticate(authenticator HttpAuthenticator) func(next http.Handler) http.Handler {
	if authenticator == nil {
		panic("authenticator cannot be nil")
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			tracer := otel.Tracer(internal.Scope)
			ctx, span := tracer.Start(r.Context(), "Authenticate")
			defer span.End()

			r = r.WithContext(ctx)

			principal, err := authenticator.Authenticate(r)
			if err != nil {
				span.RecordError(err)
				span.SetStatus(codes.Error, err.Error())
				problem := InternalServerError()
				_ = problem.Respond(w, r)
				return
			}

			span.SetAttributes(
				attribute.String("auth.principal.name", principal.Name()),
				attribute.String("auth.principal.subject_id", principal.SubjectID()),
				attribute.Bool("auth.principal.anonymous", principal.Anonymous()))

			r = r.WithContext(WithPrincipal(r.Context(), principal))
			next.ServeHTTP(w, r)
		})
	}
}

func Authenticated() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			principal, ok := PrincipalFromCtx(r.Context())
			if !ok || principal == nil || principal.Anonymous() {
				problem := Unauthorized()
				_ = problem.Respond(w, r)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func RequireRole(role string) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			principal, ok := PrincipalFromCtx(r.Context())
			if !ok || principal == nil || principal.Anonymous() {
				problem := Unauthorized()
				_ = problem.Respond(w, r)
				return
			}

			if !principal.HasRole(role) {
				problem := Forbidden()
				_ = problem.Respond(w, r)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

type nopAuthenticator struct{}

func (n nopAuthenticator) Authenticate(r *http.Request) (Principal, error) {
	return nopPrincipal{}, nil
}

type nopPrincipal struct{}

func (n nopPrincipal) Name() string {
	return ""
}

func (n nopPrincipal) SubjectID() string {
	return ""
}

func (n nopPrincipal) Anonymous() bool {
	return true
}

func (n nopPrincipal) HasRole(role string) bool {
	return false
}

func (n nopPrincipal) Attribute(key string) (any, bool) {
	return nil, false
}
