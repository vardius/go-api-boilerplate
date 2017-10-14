package middleware

import (
	"net/http"
	"time"

	"github.com/vardius/golog"
	"github.com/vardius/gorouter"
)

// NewLogger creates new logger middleware
func NewLogger(log golog.Logger) gorouter.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			log.Info(r.Context(), "[API Request|Start]: %s %q\n", r.Method, r.URL.String())
			start := time.Now()
			next.ServeHTTP(w, r)
			log.Info(r.Context(), "[API Request|End] %s %q %v\n", r.Method, r.URL.String(), time.Since(start))
		}

		return http.HandlerFunc(fn)
	}
}
