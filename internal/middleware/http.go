package middleware

import (
	"bufio"
	"io"
	"net"
	"net/http"
	"sync/atomic"
)

type ResponseWriter struct {
	http.ResponseWriter

	statusCode  int
	bytesWrote  uint64
	wroteHeader atomic.Uint32
}

func newResponseWriter(w http.ResponseWriter) *ResponseWriter {
	return &ResponseWriter{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
		bytesWrote:     0,
		wroteHeader:    atomic.Uint32{},
	}
}

func (r *ResponseWriter) Header() http.Header {
	return r.ResponseWriter.Header()
}

func (r *ResponseWriter) Write(bytes []byte) (int, error) {

	// If WriteHeader wasn't explicitly called before Respond it is implicitly called under the hood
	// with assume status code 200 OK.
	if r.wroteHeader.CompareAndSwap(0, 1) {
		r.statusCode = http.StatusOK
	}

	bytesWrote, err := r.ResponseWriter.Write(bytes)
	r.bytesWrote += uint64(bytesWrote)
	return bytesWrote, err
}

func (r *ResponseWriter) WriteHeader(statusCode int) {
	// Ensure that only the first call to WriteHeader is honored as the others are considered
	// superfluous. The HTTP headers can only be written once. This helps ensure the correct
	// status code is being captured for instrumentation.
	if r.wroteHeader.CompareAndSwap(0, 1) {
		r.statusCode = statusCode
		r.ResponseWriter.WriteHeader(statusCode)
	}
}

func (r *ResponseWriter) ReadFrom(src io.Reader) (int64, error) {
	if rf, ok := r.ResponseWriter.(io.ReaderFrom); ok {
		if r.wroteHeader.CompareAndSwap(0, 1) {
			r.statusCode = http.StatusOK
		}
		n, err := rf.ReadFrom(src)
		r.bytesWrote += uint64(n)
		return n, err
	}
	return io.Copy(&fallbackReadFromResponseWriter{parent: r}, src)
}

func (r *ResponseWriter) WriteString(s string) (int, error) {
	if ws, ok := r.ResponseWriter.(io.StringWriter); ok {
		if r.wroteHeader.CompareAndSwap(0, 1) {
			r.statusCode = http.StatusOK
		}
		n, err := ws.WriteString(s)
		r.bytesWrote += uint64(n)
		return n, err
	}
	return r.Write([]byte(s))
}

func (r *ResponseWriter) Push(target string, opts *http.PushOptions) error {
	pusher, ok := r.ResponseWriter.(http.Pusher)
	if !ok {
		return http.ErrNotSupported
	}
	return pusher.Push(target, opts)
}

func (r *ResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hijacker, ok := r.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, http.ErrNotSupported
	}
	return hijacker.Hijack()
}

func (r *ResponseWriter) Flush() {
	flusher, ok := r.ResponseWriter.(http.Flusher)
	if !ok {
		return
	}
	flusher.Flush()
}

func (r *ResponseWriter) Status() int {
	return r.statusCode
}

func (r *ResponseWriter) BytesWrote() uint64 {
	return r.bytesWrote
}

func (r *ResponseWriter) WroteHeader() bool {
	return r.wroteHeader.Load() == 1
}

func (r *ResponseWriter) Unwrap() http.ResponseWriter {
	return r.ResponseWriter
}

type fallbackReadFromResponseWriter struct {
	parent *ResponseWriter
}

func (r *fallbackReadFromResponseWriter) Write(data []byte) (int, error) {
	if r.parent.wroteHeader.CompareAndSwap(0, 1) {
		r.parent.statusCode = http.StatusOK
	}
	n, err := r.parent.ResponseWriter.Write(data)
	r.parent.bytesWrote += uint64(n)
	return n, err
}
