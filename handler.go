package yuna

import (
	"net/http"
)

// A Handler handles an HTTP request and returns a Responder.
//
// The returned Responder will be used to respond to the client by Yuna. Regardless of what occurs
// while handling the request, the Handler must return a non-nil Responder. Returning nil is not
// valid and will result in a panic.
type Handler interface {
	ServeHTTP(r *Request) Responder
}

// A HandlerFunc is an adapter to allow the use of ordinary functions as Handler.
type HandlerFunc func(r *Request) Responder

func (f HandlerFunc) ServeHTTP(r *Request) Responder {
	return f(r)
}

func wrap(h Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		responder := h.ServeHTTP(newRequest(r))
		if responder == nil {
			panic("handler returned nil responder")
		}

		err := responder.Respond(w, r)
		if err != nil {
			// todo: log error
		}
	})
}

func wrapFn(fn HandlerFunc) http.HandlerFunc {
	return wrap(fn).ServeHTTP
}
