package auth

import (
	"net/http"

	"github.com/vardius/go-api-boilerplate/pkg/auth/identity"
	"github.com/vardius/gorouter"
)

// BasicAuthFunc returns Identity from username and password combination
type BasicAuthFunc func(username, password string) (*identity.Identity, error)

// BasicAuth guard request using basic auth for authentication
func BasicAuth(afn BasicAuthFunc) gorouter.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			username, password, hasAuth := r.BasicAuth()
			if hasAuth {
				i, err := afn(username, password)
				if err != nil {
					w.Header().Set("WWW-Authenticate", `Basic`)
					http.Error(w, err.Error(), http.StatusUnauthorized)
					return
				}

				next.ServeHTTP(w, r.WithContext(identity.ContextWithIdentity(r.Context(), i)))
				return
			}

			next.ServeHTTP(w, r)
		}

		return http.HandlerFunc(fn)
	}
}
