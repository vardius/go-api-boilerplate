package middleware

import (
	"encoding/json"
	"net/http"

	"github.com/vardius/golog"
	"github.com/vardius/gorouter"
)

// NewPanicRecover creates new panic recover middleware
func NewPanicRecover(log golog.Logger) gorouter.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					log.Critical(r.Context(), "Recovered in f %v", rec)

					w.WriteHeader(http.StatusInternalServerError)
					json.NewEncoder(w).Encode(HTTPError{
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
