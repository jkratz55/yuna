package yuna

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/form/v4"
	"github.com/vmihailenco/msgpack/v5"
)

var (
	decoder = form.NewDecoder()
)

type Request struct {
	raw *http.Request
}

func newRequest(r *http.Request) *Request {
	return &Request{
		raw: r,
	}
}

func (r *Request) QueryParam(name string) ParamValue {
	q := r.raw.URL.Query()
	vals, ok := q[name]
	if !ok {
		return ParamValue{
			vals:    make([]string, 0),
			present: false,
		}
	}
	return ParamValue{
		vals:    vals,
		present: true,
	}
}

func (r *Request) PathParam(name string) ParamValue {
	val := chi.URLParamFromCtx(r.raw.Context(), name)
	if val == "" {
		return ParamValue{
			vals:    make([]string, 0),
			present: false,
		}
	}
	return ParamValue{
		vals:    []string{val},
		present: true,
	}
}

func (r *Request) ParseForm() error {
	return r.raw.ParseForm()
}

func (r *Request) Bind(dst any) error {
	_ = r.raw.ParseForm()
	return decoder.Decode(dst, r.raw.URL.Query())
}

func (r *Request) Header(name string) string {
	return r.raw.Header.Get(name)
}

func (r *Request) Context() context.Context {
	return r.raw.Context()
}

func (r *Request) Host() string {
	return r.raw.Host
}

func (r *Request) Proto() string {
	return r.raw.Proto
}

func (r *Request) ProtoMajor() int {
	return r.raw.ProtoMajor
}

func (r *Request) ProtoMinor() int {
	return r.raw.ProtoMinor
}

func (r *Request) ContentLength() int64 {
	return r.raw.ContentLength
}

func (r *Request) Method() string {
	return r.raw.Method
}

func (r *Request) RemoteAddr() string {
	return r.raw.RemoteAddr
}

func (r *Request) Body() io.ReadCloser {
	return r.raw.Body
}

func (r *Request) URL() *url.URL {
	return r.raw.URL
}

func (r *Request) Form() url.Values {
	return r.raw.Form
}

func (r *Request) FormFile(key string) (multipart.File, *multipart.FileHeader, error) {
	return r.raw.FormFile(key)
}

func (r *Request) FormValue(key string) string {
	return r.raw.FormValue(key)
}

func (r *Request) PostFormValue(key string) string {
	return r.raw.PostFormValue(key)
}

func (r *Request) PostForm() url.Values {
	return r.raw.PostForm
}

func (r *Request) MultipartForm() *multipart.Form {
	return r.raw.MultipartForm
}

func (r *Request) Trailer() http.Header {
	return r.raw.Trailer
}

func (r *Request) RequestURI() string {
	return r.raw.RequestURI
}

func (r *Request) Cookies() []*http.Cookie {
	return r.raw.Cookies()
}

func (r *Request) CookiesNamed(name string) []*http.Cookie {
	return r.raw.CookiesNamed(name)
}

func (r *Request) Cookie(name string) (*http.Cookie, error) {
	return r.raw.Cookie(name)
}

func (r *Request) MultipartReader() (*multipart.Reader, error) {
	return r.raw.MultipartReader()
}

func (r *Request) Decode(v interface{}) error {
	ct := r.raw.Header.Get("Content-Type")
	mediaType, _, _ := mime.ParseMediaType(ct)
	mediaType = strings.ToLower(mediaType)

	switch mediaType {
	case MIMEApplicationMsgpack:
		return msgpack.NewDecoder(r.raw.Body).Decode(v)
	case MIMEApplicationXML, MIMEApplicationXMLCharsetUTF8, MIMETextXML, MIMETextXMLCharsetUTF8:
		return xml.NewDecoder(r.raw.Body).Decode(v)
	default:
		// If we don't match any of the above content types, it is assumed to be JSON.
		return json.NewDecoder(r.raw.Body).Decode(v)
	}
}

func (r *Request) RawRequest() *http.Request {
	return r.raw
}
