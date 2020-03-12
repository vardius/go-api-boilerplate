package middleware

import (
	"net/http"
	"time"

	"github.com/vardius/golog"
	"github.com/vardius/gorouter/v4"

	"github.com/vardius/go-api-boilerplate/pkg/http/middleware/metadata"
)

// Logger wraps http.Handler with a logger middleware
func Logger(logger golog.Logger) gorouter.MiddlewareFunc {
	m := func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			var traceID string
			var statusCode int
			now := time.Now()

			mtd, ok := r.Context().Value(metadata.KeyMetadataValues).(*metadata.Metadata)
			if ok {
				traceID = mtd.TraceID
				statusCode = mtd.StatusCode
				now = mtd.Now
			}

			logger.Info(r.Context(), "[Request|Start]: %s : (%d) : %s %s -> %s\n",
				traceID, statusCode,
				r.Method, r.URL.Path,
				r.RemoteAddr,
			)

			next.ServeHTTP(w, r)

			logger.Info(r.Context(), "[Request|End]: %s : (%d) : %s %s -> %s (%s)\n",
				traceID, statusCode,
				r.Method, r.URL.Path,
				r.RemoteAddr, time.Since(now),
			)
		}

		return http.HandlerFunc(fn)
	}

	return m
}
