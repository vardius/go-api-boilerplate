package middleware

import (
	"errors"
	"net/http"
	"time"

	"github.com/vardius/golog"
	"github.com/vardius/gorouter/v4"

	apperrors "github.com/vardius/go-api-boilerplate/pkg/errors"
	mtd "github.com/vardius/go-api-boilerplate/pkg/metadata"
)

// Logger wraps http.Handler with a logger middleware
func Logger(logger golog.Logger) gorouter.MiddlewareFunc {
	m := func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			now := time.Now()

			logger.Info(r.Context(), "[HTTP] Start: %s %s -> %s",
				r.Method,
				r.URL.Path,
				r.RemoteAddr,
			)

			next.ServeHTTP(w, r)

			var statusCode int
			var stackTrace string
			if m, ok := mtd.FromContext(r.Context()); ok {
				statusCode = m.StatusCode

				if m.Err != nil {
					var appErr *apperrors.AppError
					if errors.As(m.Err, &appErr) {
						stackTrace, _ = appErr.StackTrace()
					} else {
						stackTrace = m.Err.Error()
					}
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
				logger.Info(r.Context(), "[HTTP] End: %s %s -> %s [%d] (%s)", args...)
			} else if statusCode != http.StatusInternalServerError {
				args = append(args, stackTrace)
				logger.Debug(r.Context(), "[HTTP] End: %s %s -> %s [%d] (%s): %s", args...)
			} else {
				args = append(args, stackTrace)
				logger.Error(r.Context(), "[HTTP] End: %s %s -> %s [%d] (%s): %s", args...)
			}
		}

		return http.HandlerFunc(fn)
	}

	return m
}
