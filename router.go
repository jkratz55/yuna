package yuna

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

type Router interface {
	http.Handler

	Use(middleware ...func(http.Handler) http.Handler)
	With(middleware ...func(http.Handler) http.Handler) Router

	Get(pattern string, fn HandlerFunc)
	Post(pattern string, fn HandlerFunc)
	Put(pattern string, fn HandlerFunc)
	Delete(pattern string, fn HandlerFunc)
	Patch(pattern string, fn HandlerFunc)
	Options(pattern string, fn HandlerFunc)
	Head(pattern string, fn HandlerFunc)
	Connect(pattern string, fn HandlerFunc)
	Trace(pattern string, fn HandlerFunc)
	Method(method, pattern string, fn Handler)

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

func (m *Mux) Get(pattern string, fn HandlerFunc) {
	m.r.Get(pattern, wrapFn(fn))
}

func (m *Mux) Post(pattern string, fn HandlerFunc) {
	m.r.Post(pattern, wrapFn(fn))
}

func (m *Mux) Put(pattern string, fn HandlerFunc) {
	m.r.Put(pattern, wrapFn(fn))
}

func (m *Mux) Delete(pattern string, fn HandlerFunc) {
	m.r.Delete(pattern, wrapFn(fn))
}

func (m *Mux) Patch(pattern string, fn HandlerFunc) {
	m.r.Patch(pattern, wrapFn(fn))
}

func (m *Mux) Options(pattern string, fn HandlerFunc) {
	m.r.Options(pattern, wrapFn(fn))
}

func (m *Mux) Head(pattern string, fn HandlerFunc) {
	m.r.Head(pattern, wrapFn(fn))
}

func (m *Mux) Connect(pattern string, fn HandlerFunc) {
	m.r.Connect(pattern, wrapFn(fn))
}

func (m *Mux) Trace(pattern string, fn HandlerFunc) {
	m.r.Trace(pattern, wrapFn(fn))
}

func (m *Mux) Method(method string, pattern string, handler Handler) {
	m.r.Method(method, pattern, wrap(handler))
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
