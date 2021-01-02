package middleware

import (
	"net/http"

	httpcors "github.com/rs/cors"
	"github.com/vardius/gorouter/v4"

	"github.com/vardius/go-api-boilerplate/pkg/identity"
)

var (
	allowedMethods = []string{
		http.MethodHead,
		http.MethodGet,
		http.MethodPost,
		http.MethodPut,
		http.MethodDelete,
	}
	allowedHeaders = []string{"*"}
)

// CORS replies to request with cors header and handles preflight request
// it is enhancement to improve middleware usability instead of wrapping every handler
func CORS(allowedOrigins []string, debug bool) gorouter.MiddlewareFunc {
	defaultCors := httpcors.New(httpcors.Options{
		AllowCredentials: true,
		AllowedOrigins:   allowedOrigins,
		AllowedMethods:   allowedMethods,
		AllowedHeaders:   allowedHeaders,
		Debug:            debug,
	})

	m := func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			if i, isAuthorized := identity.FromContext(r.Context()); isAuthorized {
				if i.ClientDomain != "" {
					cors := httpcors.New(httpcors.Options{
						AllowCredentials: true,
						AllowedOrigins:   []string{i.ClientDomain},
						AllowedMethods:   allowedMethods,
						AllowedHeaders:   allowedHeaders,
						Debug:            debug,
					})

					cors.Handler(next).ServeHTTP(w, r)

					return
				}
			}

			defaultCors.Handler(next).ServeHTTP(w, r)
		}

		return http.HandlerFunc(fn)
	}

	return m
}
