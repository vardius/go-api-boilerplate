package middleware

import (
	"net/http"

	"github.com/vardius/gorouter"
)

// NewCors creates cors middleware
func NewCors(origins []string) gorouter.MiddlewareFunc {
	whitelist := make(map[string]bool)
	for _, v := range origins {
		whitelist[v] = true
	}

	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			//Verify that request has an origin header
			if origin == "" {
				http.Error(w, "Cross domain requests require Origin header", http.StatusBadRequest)
				return
			}

			if whitelist[origin] != true {
				http.Error(w, "Origin not allowed", http.StatusBadRequest)
				return
			}

			next.ServeHTTP(w, r)
		}

		return http.HandlerFunc(fn)
	}
}
