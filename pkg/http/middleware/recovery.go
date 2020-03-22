package middleware

import (
	"net/http"
	"runtime/debug"

	"github.com/vardius/golog"
	"github.com/vardius/gorouter/v4"

	"github.com/vardius/go-api-boilerplate/pkg/errors"
	"github.com/vardius/go-api-boilerplate/pkg/http/response"
)

// Recover middleware recovers from panic
func Recover(logger golog.Logger) gorouter.MiddlewareFunc {
	m := func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					logger.Critical(r.Context(), "[HTTP] Recovered in %v\n%s\n", rec, debug.Stack())

					appErr := errors.New(errors.INTERNAL, http.StatusText(http.StatusInternalServerError))
					response.WriteHeader(r.Context(), w, errors.HTTPStatusCode(appErr))

					if err := response.JSON(r.Context(), w, appErr); err != nil {
						logger.Critical(r.Context(), "[HTTP] Errors while sending response after panic %v\n", err)
					}
				}
			}()

			next.ServeHTTP(w, r)
		}

		return http.HandlerFunc(fn)
	}

	return m
}
