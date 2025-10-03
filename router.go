package yuna

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

type Router interface {
	http.Handler

	Use(middleware ...func(http.Handler) http.Handler)
	With(middleware ...func(http.Handler) http.Handler) Router

	Get(pattern string, fn HandlerFunc, middleware ...HttpMiddleware)
	Post(pattern string, fn HandlerFunc, middleware ...HttpMiddleware)
	Put(pattern string, fn HandlerFunc, middleware ...HttpMiddleware)
	Delete(pattern string, fn HandlerFunc, middleware ...HttpMiddleware)
	Patch(pattern string, fn HandlerFunc, middleware ...HttpMiddleware)
	Options(pattern string, fn HandlerFunc, middleware ...HttpMiddleware)
	Head(pattern string, fn HandlerFunc, middleware ...HttpMiddleware)
	Connect(pattern string, fn HandlerFunc, middleware ...HttpMiddleware)
	Trace(pattern string, fn HandlerFunc, middleware ...HttpMiddleware)
	Method(method, pattern string, fn Handler, middleware ...HttpMiddleware)

	Mount(pattern string, h http.Handler)
	Route(pattern string, fn func(r Router))
	Group(fn func(r Router))
}

type Mux struct {
	r chi.Router
}

func NewMux() *Mux {
	return &Mux{
		r: chi.NewRouter(),
	}
}

func (m *Mux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	m.r.ServeHTTP(w, r)
}

func (m *Mux) Use(middleware ...func(http.Handler) http.Handler) {
	m.r.Use(middleware...)
}

func (m *Mux) With(middleware ...func(http.Handler) http.Handler) Router {
	router := m.r.With(middleware...)
	return &Mux{r: router}
}

func (m *Mux) Get(pattern string, fn HandlerFunc, middleware ...HttpMiddleware) {
	m.r.Get(pattern, wrapFn(fn, middleware...))
}

func (m *Mux) Post(pattern string, fn HandlerFunc, middleware ...HttpMiddleware) {
	m.r.Post(pattern, wrapFn(fn, middleware...))
}

func (m *Mux) Put(pattern string, fn HandlerFunc, middleware ...HttpMiddleware) {
	m.r.Put(pattern, wrapFn(fn, middleware...))
}

func (m *Mux) Delete(pattern string, fn HandlerFunc, middleware ...HttpMiddleware) {
	m.r.Delete(pattern, wrapFn(fn, middleware...))
}

func (m *Mux) Patch(pattern string, fn HandlerFunc, middleware ...HttpMiddleware) {
	m.r.Patch(pattern, wrapFn(fn, middleware...))
}

func (m *Mux) Options(pattern string, fn HandlerFunc, middleware ...HttpMiddleware) {
	m.r.Options(pattern, wrapFn(fn, middleware...))
}

func (m *Mux) Head(pattern string, fn HandlerFunc, middleware ...HttpMiddleware) {
	m.r.Head(pattern, wrapFn(fn, middleware...))
}

func (m *Mux) Connect(pattern string, fn HandlerFunc, middleware ...HttpMiddleware) {
	m.r.Connect(pattern, wrapFn(fn, middleware...))
}

func (m *Mux) Trace(pattern string, fn HandlerFunc, middleware ...HttpMiddleware) {
	m.r.Trace(pattern, wrapFn(fn, middleware...))
}

func (m *Mux) Method(method string, pattern string, handler Handler, middleware ...HttpMiddleware) {
	m.r.Method(method, pattern, wrap(handler, middleware...))
}

func (m *Mux) Mount(pattern string, h http.Handler) {
	m.r.Mount(pattern, h)
}

func (m *Mux) Route(pattern string, fn func(r Router)) {
	m.r.Route(pattern, func(r chi.Router) {
		fn(&Mux{r: r})
	})
}

func (m *Mux) Group(fn func(r Router)) {
	m.r.Group(func(r chi.Router) {
		fn(&Mux{r: r})
	})
}
