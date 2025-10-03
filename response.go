package yuna

import (
	"bytes"
	"encoding/json"
	"html/template"
	"net/http"
)

// A Responder responds to an HTTP request.
type Responder interface {
	Respond(w http.ResponseWriter, r *http.Request) error
}

// A ResponderFunc is an adapter to allow the use of ordinary functions as Responder.
type ResponderFunc func(w http.ResponseWriter, r *http.Request) error

func (f ResponderFunc) Respond(w http.ResponseWriter, r *http.Request) error {
	return f(w, r)
}

type ResponseBuilder struct {
	status  int
	header  http.Header
	cookies []*http.Cookie
	body    any
	html    bool
}

func Response() *ResponseBuilder {
	return &ResponseBuilder{
		status:  0,
		header:  http.Header{},
		cookies: make([]*http.Cookie, 0),
		body:    nil,
		html:    false,
	}
}

func (rb *ResponseBuilder) Status(status int) *ResponseBuilder {
	rb.status = status
	return rb
}

func (rb *ResponseBuilder) Header(key string, values ...string) *ResponseBuilder {
	for _, value := range values {
		rb.header.Add(key, value)
	}
	return rb
}

func (rb *ResponseBuilder) Headers(headers http.Header) *ResponseBuilder {
	for key, values := range headers {
		for _, value := range values {
			rb.header.Add(key, value)
		}
	}
	return rb
}

func (rb *ResponseBuilder) Cookie(cookie *http.Cookie) *ResponseBuilder {
	rb.cookies = append(rb.cookies, cookie)
	return rb
}

func (rb *ResponseBuilder) Cookies(cookies []*http.Cookie) *ResponseBuilder {
	rb.cookies = append(rb.cookies, cookies...)
	return rb
}

func (rb *ResponseBuilder) Body(body any) *ResponseBuilder {
	rb.body = body
	return rb
}

func (rb *ResponseBuilder) Html(tpl *template.Template, name string, data any) *ResponseBuilder {
	var buf bytes.Buffer
	var err error

	if name == "" {
		err = tpl.Execute(&buf, data)
	} else {
		err = tpl.ExecuteTemplate(&buf, name, data)
	}
	if err != nil {
		panic(err)
	}

	rb.body = buf.Bytes()
	rb.html = true
	return rb
}

func (rb *ResponseBuilder) Respond(w http.ResponseWriter, r *http.Request) error {

	// todo: only supporting JSON and HTML for now, but we should handle content negotiation

	if rb.status == 0 && rb.body != nil {
		rb.status = http.StatusOK
	}
	if rb.status == 0 && rb.body == nil {
		rb.status = http.StatusNoContent
	}

	// Transpose headers to ResponseWriter
	for key, values := range rb.header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	for _, cookie := range rb.cookies {
		http.SetCookie(w, cookie)
	}

	if rb.html {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
	} else {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
	}

	w.WriteHeader(rb.status)

	if rb.body == nil {
		return nil
	}

	return json.NewEncoder(w).Encode(rb.body)
}

func Ok(body any) *ResponseBuilder {
	return &ResponseBuilder{
		status:  http.StatusOK,
		header:  http.Header{},
		body:    body,
		cookies: make([]*http.Cookie, 0),
	}
}

func Created(location string) *ResponseBuilder {
	return &ResponseBuilder{
		status: http.StatusCreated,
		header: http.Header{
			"Location": []string{location},
		},
		body:    nil,
		cookies: make([]*http.Cookie, 0),
	}
}

func Accepted() *ResponseBuilder {
	return &ResponseBuilder{
		status:  http.StatusAccepted,
		header:  http.Header{},
		body:    nil,
		cookies: make([]*http.Cookie, 0),
	}
}

func NoContent() *ResponseBuilder {
	return &ResponseBuilder{
		status:  http.StatusNoContent,
		header:  http.Header{},
		body:    nil,
		cookies: make([]*http.Cookie, 0),
	}
}
