package middleware

import (
	"expvar"
	"net/http"
	"runtime"

	"github.com/vardius/gorouter/v4"
)

var hasMetrics bool

// Metrics updates program counters.
func Metrics() gorouter.MiddlewareFunc {
	hasMetrics = true
	goroutines := expvar.NewInt("goroutines")
	requests := expvar.NewInt("http_requests")

	m := func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r)

			// Increment the request counter.
			requests.Add(1)

			// Update the count for the number of active goroutines every 100 requests.
			if requests.Value()%100 == 0 {
				goroutines.Set(int64(runtime.NumGoroutine()))
			}
		}

		return http.HandlerFunc(fn)
	}

	return m
}
