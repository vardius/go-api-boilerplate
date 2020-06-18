package middleware

import (
	"expvar"
	"net/http"
	"runtime"

	"github.com/vardius/gorouter/v4"
)

// m contains the global program counters for the application.
var m = struct {
	gr  *expvar.Int
	req *expvar.Int
	rl  *expvar.Map
}{
	gr:  expvar.NewInt("goroutines"),
	req: expvar.NewInt("requests"),
	rl:  expvar.NewMap("rateLimits"),
}

// Metrics updates program counters.
func Metrics() gorouter.MiddlewareFunc {
	m := func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r)

			// Increment the request counter.
			m.req.Add(1)

			// Update the count for the number of active goroutines every 100 requests.
			if m.req.Value()%100 == 0 {
				m.gr.Set(int64(runtime.NumGoroutine()))
			}
		}

		return http.HandlerFunc(fn)
	}

	return m
}
