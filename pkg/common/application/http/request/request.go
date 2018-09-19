package request

import (
	"net/http"
)

// LimitRequestBody limits the request body
func LimitRequestBody(n int64) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			r.Body = http.MaxBytesReader(w, r.Body, n)

			next.ServeHTTP(w, r)
		}

		return http.HandlerFunc(fn)
	}
}
