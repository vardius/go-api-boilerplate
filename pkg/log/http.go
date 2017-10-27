package log

import (
	"net/http"
	"time"
)

// LogRequest wraps http.Handler with a logger middleware
func (l *Logger) LogRequest(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		l.Info(r.Context(), "[API Request|Start]: %s %q\n", r.Method, r.URL.String())
		start := time.Now()
		next.ServeHTTP(w, r)
		l.Info(r.Context(), "[API Request|End] %s %q %v\n", r.Method, r.URL.String(), time.Since(start))
	}

	return http.HandlerFunc(fn)
}
