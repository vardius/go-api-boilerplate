package auth

import (
	"encoding/base64"
	"net/http"
	"strings"

	"github.com/vardius/go-api-boilerplate/pkg/auth/identity"
	"github.com/vardius/gorouter"
)

// TokenAuthFunc returns Identity from auth token
type TokenAuthFunc func(token string) (*identity.Identity, error)

// Bearer guard request using bearer token for authentication
func Bearer(realm string, afn TokenAuthFunc) gorouter.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			token := r.Header.Get("Authorization")
			if token == "" {
				next.ServeHTTP(w, r)
				return
			}

			if strings.HasPrefix(token, "Bearer ") {
				if bearer, err := base64.StdEncoding.DecodeString(token[7:]); err == nil {
					i, err := afn(string(bearer))
					if err != nil {
						w.Header().Set("WWW-Authenticate", `Bearer realm="`+realm+`"`)
						http.Error(w, err.Error(), http.StatusUnauthorized)
						return
					}

					next.ServeHTTP(w, r.WithContext(identity.ContextWithIdentity(r.Context(), i)))
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

// Query guard request using query token for authentication
func Query(name string, afn TokenAuthFunc) gorouter.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			token := r.URL.Query().Get(name)
			if token == "" {
				next.ServeHTTP(w, r)
				return
			}

			i, err := afn(token)
			if err != nil {
				http.Error(w, err.Error(), http.StatusUnauthorized)
				return
			}

			next.ServeHTTP(w, r.WithContext(identity.ContextWithIdentity(r.Context(), i)))
		}

		return http.HandlerFunc(fn)
	}
}

// Cookie guard request using token for authentication
func Cookie(name string, afn TokenAuthFunc) gorouter.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie(name)
			if err != nil {
				next.ServeHTTP(w, r)
				return
			}

			i, err := afn(cookie.Value)
			if err != nil {
				http.Error(w, err.Error(), http.StatusUnauthorized)
				return
			}

			next.ServeHTTP(w, r.WithContext(identity.ContextWithIdentity(r.Context(), i)))
		}

		return http.HandlerFunc(fn)
	}
}
