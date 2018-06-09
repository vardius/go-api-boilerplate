package response

import "net/http"

// WithXSS sets xss response header types
func WithXSS(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		// Add X-Content-Type-Options header
		w.Header().Add("X-Content-Type-Options", "nosniff")
		// Prevent page from being displayed in an iframe
		w.Header().Add("X-Frame-Options", "DENY")

		next.ServeHTTP(w, r)
	}

	return http.HandlerFunc(fn)
}
