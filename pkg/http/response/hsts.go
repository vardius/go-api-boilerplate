package response

import "net/http"

// HSTS HTTP Strict Transport Security
// is an opt-in security enhancement that is specified by a web application
// through the use of a special response header
func HSTS(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		// Add Strict-Transport-Security header
		w.Header().Add("Strict-Transport-Security", "max-age=63072000; includeSubDomains")

		next.ServeHTTP(w, r)
	}

	return http.HandlerFunc(fn)
}
