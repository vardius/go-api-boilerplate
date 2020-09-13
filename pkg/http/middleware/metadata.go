package middleware

import (
	"net/http"

	"github.com/vardius/gorouter/v4"

	"github.com/vardius/go-api-boilerplate/pkg/http/request"
	md "github.com/vardius/go-api-boilerplate/pkg/metadata"
)

// responseWriter is a minimal wrapper for http.ResponseWriter that allows the
// written HTTP statusCode to be captured for metadata.
type responseWriter struct {
	http.ResponseWriter
	mtd         *md.Metadata
	wroteHeader bool
}

func wrapResponseWriter(w http.ResponseWriter, mtd *md.Metadata) *responseWriter {
	return &responseWriter{ResponseWriter: w, mtd: mtd}
}

func (rw *responseWriter) WriteHeader(statusCode int) {
	if rw.wroteHeader {
		return
	}

	rw.mtd.StatusCode = statusCode
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

			mtd.RemoteAddr = r.RemoteAddr
			mtd.UserAgent = r.UserAgent()
			mtd.Referer = r.Referer()
			if ip, err := request.IpAddress(r); err == nil {
				mtd.IPAddress = ip
			}

			ctx := md.ContextWithMetadata(r.Context(), mtd)

			next.ServeHTTP(wrapResponseWriter(w, mtd), r.WithContext(ctx))
		}

		return http.HandlerFunc(fn)
	}

	return m
}
