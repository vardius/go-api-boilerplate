package middleware

import (
	"net/http"

	"github.com/vardius/gorouter/v4"
)

// HSTS HTTP Strict Transport Security
// is an opt-in security enhancement that is specified by a web application
// through the use of a special response header
func HSTS() gorouter.MiddlewareFunc {
	m := func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			// Add Strict-Transport-Security header
			w.Header().Add("Strict-Transport-Security", "max-age=63072000; includeSubDomains")

			next.ServeHTTP(w, r)
		}

		return http.HandlerFunc(fn)
	}

	return m
}
