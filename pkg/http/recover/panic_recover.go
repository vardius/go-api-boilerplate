package recover

import (
	"encoding/json"
	"net/http"

	"github.com/vardius/go-api-boilerplate/pkg/http/response"
	"github.com/vardius/golog"
)

// A Recover recovers http handler from panic
type Recover func(next http.Handler) http.Handler

// New creates new panic recover middleware
func New() Recover {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					w.WriteHeader(http.StatusInternalServerError)
					json.NewEncoder(w).Encode(response.HTTPError{
						Code:    http.StatusInternalServerError,
						Message: http.StatusText(http.StatusInternalServerError),
					})
				}
			}()

			next.ServeHTTP(w, r)
		}

		return http.HandlerFunc(fn)
	}
}

// WithLogger returns which deffered panic will be also logged
func WithLogger(log golog.Logger) Recover {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					log.Critical(r.Context(), "Recovered in f %v", rec)

					w.WriteHeader(http.StatusInternalServerError)
					json.NewEncoder(w).Encode(response.HTTPError{
						Code:    http.StatusInternalServerError,
						Message: http.StatusText(http.StatusInternalServerError),
					})
				}
			}()

			next.ServeHTTP(w, r)
		}

		return http.HandlerFunc(fn)
	}
}
