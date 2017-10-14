package middleware

import (
	"app/pkg/auth"
	"encoding/base64"
	"net/http"
	"strings"

	"github.com/vardius/gorouter"
)

// BasicAuth guard request using basic auth for authentication
func BasicAuth(afn auth.BasicAuthFunc) gorouter.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			username, password, hasAuth := r.BasicAuth()
			if hasAuth {
				identity, err := afn(username, password)
				if err != nil {
					http.Error(w, err.Error(), http.StatusUnauthorized)
					return
				}

				next.ServeHTTP(w, r.WithContext(auth.NewContext(r, identity)))
				return
			}

			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		}

		return http.HandlerFunc(fn)
	}
}

// Bearer guard request using basic bearer token for authentication
func Bearer(realm string, afn auth.TokenAuthFunc) gorouter.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			token := r.Header.Get("Authorization")
			if strings.HasPrefix(token, "Bearer ") {
				if bearer, err := base64.StdEncoding.DecodeString(token[7:]); err == nil {
					identity, err := afn(string(bearer))
					if err != nil {
						w.Header().Set("WWW-Authenticate", `Bearer realm="`+realm+`"`)
						http.Error(w, err.Error(), http.StatusUnauthorized)
						return
					}

					next.ServeHTTP(w, r.WithContext(auth.NewContext(r, identity)))
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
func Query(name string, afn auth.TokenAuthFunc) gorouter.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			token := r.URL.Query().Get(name)

			identity, err := afn(token)
			if err != nil {
				http.Error(w, err.Error(), http.StatusUnauthorized)
				return
			}

			next.ServeHTTP(w, r.WithContext(auth.NewContext(r, identity)))
		}

		return http.HandlerFunc(fn)
	}
}
