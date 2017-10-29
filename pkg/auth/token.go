package auth

import (
	"app/pkg/auth/identity"
	"encoding/base64"
	"net/http"
	"strings"

	"github.com/vardius/gorouter"
)

// TokenAuthFunc returns Identity from auth token
type TokenAuthFunc func(token string) (*identity.Identity, error)

// Bearer guard request using basic bearer token for authentication
func Bearer(realm string, afn TokenAuthFunc) gorouter.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			token := r.Header.Get("Authorization")
			if strings.HasPrefix(token, "Bearer ") {
				if bearer, err := base64.StdEncoding.DecodeString(token[7:]); err == nil {
					i, err := afn(string(bearer))
					if err != nil {
						w.Header().Set("WWW-Authenticate", `Bearer realm="`+realm+`"`)
						http.Error(w, err.Error(), http.StatusUnauthorized)
						return
					}

					next.ServeHTTP(w, r.WithContext(identity.NewContext(r, i)))
					return
				}
			}

			w.Header().Set("WWW-Authenticate", `Bearer realm="`+realm+`"`)
			http.Error(w,
				http.StatusText(http.StatusUnauthorized),
				http.StatusUnauthorized,
			)
		}

		return http.HandlerFunc(fn)
	}
}

// Query guard request using basic query token for authentication
func Query(name string, afn TokenAuthFunc) gorouter.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			token := r.URL.Query().Get(name)

			i, err := afn(token)
			if err != nil {
				http.Error(w, err.Error(), http.StatusUnauthorized)
				return
			}

			next.ServeHTTP(w, r.WithContext(identity.NewContext(r, i)))
		}

		return http.HandlerFunc(fn)
	}
}
