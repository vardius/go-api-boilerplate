package middleware

import (
	"net/http"
	"time"

	"github.com/vardius/go-api-boilerplate/internal/http/middleware/metadata"
	"github.com/vardius/golog"
	gorouter "github.com/vardius/gorouter/v4"
)

// Logger wraps http.Handler with a logger middleware
func Logger(logger golog.Logger) gorouter.MiddlewareFunc {
	m := func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			var traceID string
			var statusCode int
			now := time.Now()

			metadata, ok := r.Context().Value(metadata.KeyMetadataValues).(*metadata.Metadata)
			if ok {
				traceID = metadata.TraceID
				statusCode = metadata.StatusCode
				now = metadata.Now
			}

			logger.Info(r.Context(), "[Request|Start]: %s : (%d) : %s %s -> %s (%s)",
				traceID, statusCode,
				r.Method, r.URL.Path,
				r.RemoteAddr, now,
			)

			next.ServeHTTP(w, r)

			logger.Info(r.Context(), "[Request|Start]: %s : (%d) : %s %s -> %s (%s)",
				traceID, statusCode,
				r.Method, r.URL.Path,
				r.RemoteAddr, time.Since(now),
			)
		}

		return http.HandlerFunc(fn)
	}

	return m
}
