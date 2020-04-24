package middleware

import (
	"net/http"

	"github.com/vardius/gorouter/v4"

	md "github.com/vardius/go-api-boilerplate/pkg/metadata"
)

// responseWriter is a minimal wrapper for http.ResponseWriter that allows the
// written HTTP statusCode to be captured for metadata.
type responseWriter struct {
	http.ResponseWriter
	statusCode  int
	wroteHeader bool
}

func wrapResponseWriter(w http.ResponseWriter) *responseWriter {
	return &responseWriter{ResponseWriter: w}
}

func (rw *responseWriter) Status() int {
	return rw.statusCode
}

func (rw *responseWriter) WriteHeader(statusCode int) {
	if rw.wroteHeader {
		return
	}

	rw.statusCode = statusCode
	rw.ResponseWriter.WriteHeader(statusCode)
	rw.wroteHeader = true

	return
}

// WithMetadata adds Metadata to requests context
func WithMetadata() gorouter.MiddlewareFunc {
	m := func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			// set the context with the required values to
			// process the request.
			mtd := md.New()

			ctx := md.ContextWithMetadata(r.Context(), mtd)

			wrapped := wrapResponseWriter(w)

			next.ServeHTTP(wrapped, r.WithContext(ctx))

			mtd.StatusCode = wrapped.Status()
		}

		return http.HandlerFunc(fn)
	}

	return m
}
