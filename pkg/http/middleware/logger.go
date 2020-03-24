package middleware

import (
	"net/http"
	"time"

	"github.com/vardius/gorouter/v4"

	"github.com/vardius/go-api-boilerplate/pkg/log"
	"github.com/vardius/go-api-boilerplate/pkg/metadata"
)

// Logger wraps http.Handler with a logger middleware
func Logger(logger *log.Logger) gorouter.MiddlewareFunc {
	m := func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			var traceID string
			var statusCode int
			now := time.Now()

			mtd, ok := metadata.FromContext(r.Context())
			if ok {
				traceID = mtd.TraceID
				statusCode = mtd.StatusCode
				now = mtd.Now
			}

			logger.Info(r.Context(), "[HTTP] Start: %s : (%d) : %s %s -> %s\n",
				traceID, statusCode,
				r.Method, r.URL.Path,
				r.RemoteAddr,
			)

			next.ServeHTTP(w, r)

			logger.Info(r.Context(), "[HTTP] End: %s : (%d) : %s %s -> %s (%s)\n",
				traceID, statusCode,
				r.Method, r.URL.Path,
				r.RemoteAddr, time.Since(now),
			)
		}

		return http.HandlerFunc(fn)
	}

	return m
}
