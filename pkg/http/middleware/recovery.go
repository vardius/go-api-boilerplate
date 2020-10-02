package middleware

import (
	"fmt"
	"net/http"
	"runtime/debug"

	"github.com/vardius/golog"
	"github.com/vardius/gorouter/v4"

	"github.com/vardius/go-api-boilerplate/pkg/application"
	apperrors "github.com/vardius/go-api-boilerplate/pkg/errors"
	"github.com/vardius/go-api-boilerplate/pkg/http/response"
)

// Recover middleware recovers from panic
func Recover(logger golog.Logger) gorouter.MiddlewareFunc {
	m := func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					logger.Critical(r.Context(), "[HTTP] Recovered in %v %s", rec, debug.Stack())

					appErr := apperrors.Wrap(fmt.Errorf("%w: recovered from panic", application.ErrInternal))

					if err := response.JSONError(r.Context(), w, appErr); err != nil {
						logger.Critical(r.Context(), "[HTTP] Errors while sending response after panic %v", err)
					}
				}
			}()

			next.ServeHTTP(w, r)
		}

		return http.HandlerFunc(fn)
	}

	return m
}
