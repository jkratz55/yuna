package yuna

import (
	"mime"
	"net/http"
	"strings"
)

const (
	CharsetUTF8 = "charset=utf-8"
)

const (
	MIMEApplicationJSON                  = "application/json"
	MIMEApplicationJSONCharsetUTF8       = MIMEApplicationJSON + "; " + CharsetUTF8
	MIMEApplicationJavascript            = "application/javascript"
	MIMEApplicationJavascriptCharsetUTF8 = MIMEApplicationJavascript + "; " + CharsetUTF8
	MIMEApplicationXML                   = "application/xml"
	MIMEApplicationXMLCharsetUTF8        = MIMEApplicationXML + "; " + CharsetUTF8
	MIMEApplicationForm                  = "application/x-www-form-urlencoded"
	MIMEApplicationProtobuf              = "application/protobuf"
	MIMEApplicationMsgpack               = "application/msgpack"
	MIMETextXML                          = "text/xml"
	MIMETextXMLCharsetUTF8               = MIMETextXML + "; " + CharsetUTF8
	MIMETextHTML                         = "text/html"
	MIMETextHTMLCharsetUTF8              = MIMETextHTML + "; " + CharsetUTF8
	MIMETextPlain                        = "text/plain"
	MIMETextPlainCharsetUTF8             = MIMETextPlain + "; " + CharsetUTF8
	MIMEMultipartForm                    = "multipart/form-data"
	MIMEOctetStream                      = "application/octet-stream"
)

// Consumes returns an HTTP middleware that inspects the Content-Type of the request body is one that
// the server/application understands/supports. If the Content-Type of the request is not supported,
// or if the Content-Type is missing, the middleware will respond with an HTTP 415 Unsupported Media
// Type response.
//
// The allowed types are specified as a list of strings. Each string should be a valid MIME type.
// The allowed types may include wildcards, for example:
//
//	"application/json", "application/*", "application/vnd.company+json"
//
// Note that providing no allowed types is effectively a no-op and all requests will be allowed.
//
// The Content-Type is not inspected for GET, HEAD, DELETE, OPTIONS requests as those request methods
// are not expected to have a body. Request with those methods will always be allowed.
func Consumes(allowed ...string) HttpMiddleware {

	// If no allowed types are specified, all will be allowed.
	if len(allowed) == 0 {
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				next.ServeHTTP(w, r)
			})
		}
	}

	allowed = normalizeMediaTypes(allowed)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			// Technically GET, HEAD, DELETE, OPTIONS can have a body, the HTTP spec doesn't have defined
			// semantics for them. The means servers are not obligated to understand or process the body,
			// and many implementations may ignore the body or reject the request. The Consumes middleware
			// simply won't check the content type for these methods because they are not expected to have
			// a body.
			if r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodDelete ||
				r.Method == http.MethodOptions {
				next.ServeHTTP(w, r)
				return
			}

			ct := r.Header.Get(HeaderContentType)
			mediatype, _, err := mime.ParseMediaType(ct)
			if err != nil || mediatype == "" {
				// If the method can have a body, but the content type is not specified or is invalid,
				// return an HTTP 415 Unsupported Media Type response.
				prob := UnsupportedMediaType()
				prob.ServeHTTP(w, r)
				return
			}

			if matchesAny(mediatype, allowed) {
				next.ServeHTTP(w, r)
				return
			}

			prob := UnsupportedMediaType()
			prob.ServeHTTP(w, r)
		})
	}
}

func normalizeMediaTypes(types []string) []string {
	out := make([]string, 0, len(types))
	seen := map[string]struct{}{}
	for _, t := range types {
		mt := strings.ToLower(strings.TrimSpace(t))
		if mt == "" {
			continue
		}
		if _, ok := seen[mt]; !ok {
			seen[mt] = struct{}{}
			out = append(out, mt)
		}
	}
	return out
}

func matchesAny(mt string, allowed []string) bool {
	mt = strings.ToLower(mt)
	for _, a := range allowed {
		if mt == a {
			return true
		}
		// support type/* wildcards in allowed list
		if strings.HasSuffix(a, "/*") {
			prefix := strings.SplitN(a, "/", 2)[0]
			if strings.HasPrefix(mt, prefix+"/") {
				return true
			}
		}
	}
	return false
}
