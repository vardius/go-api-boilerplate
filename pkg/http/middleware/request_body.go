package middleware

import (
	"net/http"

	"github.com/vardius/gorouter/v4"
)

// LimitRequestBody limits the request body
func LimitRequestBody(n int64) gorouter.MiddlewareFunc {
	m := func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			r.Body = http.MaxBytesReader(w, r.Body, n)

			next.ServeHTTP(w, r)
		}

		return http.HandlerFunc(fn)
	}

	return m
}
