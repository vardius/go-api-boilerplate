package middleware

import (
	"net/http"
	"time"

	"github.com/vardius/gorouter/v4"

	"github.com/vardius/go-api-boilerplate/pkg/container"
	"github.com/vardius/go-api-boilerplate/pkg/log"
	mtd "github.com/vardius/go-api-boilerplate/pkg/metadata"
)

// Logger wraps http.Handler with a logger middleware
func Logger(logger *log.Logger) gorouter.MiddlewareFunc {
	m := func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			now := time.Now()

			logger.Info(r.Context(), "[HTTP] Start: %s %s -> %s",
				r.Method, r.URL.Path,
				r.RemoteAddr,
			)

			if requestContainer, ok := container.FromContext(r.Context()); ok {
				requestContainer.Register("logger", logger)
			}

			next.ServeHTTP(w, r)

			var statusCode int
			if m, ok := mtd.FromContext(r.Context()); ok {
				statusCode = m.StatusCode
			}

			logger.Info(r.Context(), "[HTTP] End: %s %s -> %s [%d] (%s)",
				r.Method, r.URL.Path,
				r.RemoteAddr,
				statusCode,
				time.Since(now),
			)
		}

		return http.HandlerFunc(fn)
	}

	return m
}
