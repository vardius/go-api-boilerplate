package middleware

import (
	"fmt"
	"net/http"
	"time"

	"github.com/vardius/go-api-boilerplate/pkg/logger"
	mtd "github.com/vardius/go-api-boilerplate/pkg/metadata"
	"github.com/vardius/gorouter/v4"
)

// Logger wraps http.Handler with a logger middleware
func Logger() gorouter.MiddlewareFunc {
	m := func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			now := time.Now()

			logger.Info(r.Context(), fmt.Sprintf("[HTTP] Start: %s %s -> %s",
				r.Method,
				r.URL.Path,
				r.RemoteAddr,
			))

			next.ServeHTTP(w, r)

			var statusCode int
			var stackTrace string
			if m, ok := mtd.FromContext(r.Context()); ok {
				statusCode = m.StatusCode
				statusCode = m.StatusCode
				if m.Err != nil {
					stackTrace = m.Err.Error()
				}
			}

			args := []interface{}{
				r.Method,
				r.URL.Path,
				r.RemoteAddr,
				statusCode,
				time.Since(now),
			}

			if stackTrace == "" {
				logger.Info(r.Context(), fmt.Sprintf("[HTTP] End: %s %s -> %s [%d] (%s)", args...))
			} else if statusCode != http.StatusInternalServerError {
				args = append(args, stackTrace)
				logger.Debug(r.Context(), fmt.Sprintf("[HTTP] End: %s %s -> %s [%d] (%s): %s", args...))
			} else {
				args = append(args, stackTrace)
				logger.Error(r.Context(), fmt.Sprintf("[HTTP] End: %s %s -> %s [%d] (%s): %s", args...))
			}
		}

		return http.HandlerFunc(fn)
	}

	return m
}
