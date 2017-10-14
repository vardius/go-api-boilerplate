package middleware

import (
	"net/http"
)

// JSONHeader sets json response header type
func JSONHeader(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		next.ServeHTTP(w, r)
	}

	return http.HandlerFunc(fn)
}

// XSSHeader sets xss response header types
func XSSHeader(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		// Add X-Content-Type-Options header
		w.Header().Add("X-Content-Type-Options", "nosniff")
		// Prevent page from being displayed in an iframe
		w.Header().Add("X-Frame-Options", "DENY")

		next.ServeHTTP(w, r)
	}

	return http.HandlerFunc(fn)
}
