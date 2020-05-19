package authenticator

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"

	"github.com/vardius/go-api-boilerplate/pkg/application"
	"github.com/vardius/go-api-boilerplate/pkg/http/response"
	"github.com/vardius/go-api-boilerplate/pkg/identity"
)

// TokenAuthFunc returns Identity from token
type TokenAuthFunc func(token string) (identity.Identity, error)

// TokenAuthenticator authorize by token
// and adds Identity to request's Context
type TokenAuthenticator interface {
	// FromHeader authorize by the token provided in the request's Authorization header
	FromHeader(realm string) func(next http.Handler) http.Handler
	// FromQuery authorize by the token provided in the request's query parameter
	FromQuery(name string) func(next http.Handler) http.Handler
	// FromCookie authorize by the token provided in the request's cookie
	FromCookie(name string) func(next http.Handler) http.Handler
}

type tokenAuth struct {
	afn TokenAuthFunc
}

func (a *tokenAuth) FromHeader(realm string) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			token := r.Header.Get("Authorization")
			if token == "" {
				next.ServeHTTP(w, r)
				return
			}

			if strings.HasPrefix(token, "Bearer ") {
				if bearer, err := base64.StdEncoding.DecodeString(token[7:]); err == nil {
					i, err := a.afn(string(bearer))
					if err != nil {
						w.Header().Set("WWW-Authenticate", `Bearer realm="`+realm+`"`)

						response.MustJSONError(r.Context(), w, fmt.Errorf("%w: %s", application.ErrUnauthorized, err))
						return
					}

					next.ServeHTTP(w, r.WithContext(identity.ContextWithIdentity(r.Context(), i)))
					return
				}
			}

			w.Header().Set("WWW-Authenticate", `Bearer realm="`+realm+`"`)

			response.MustJSONError(r.Context(), w, application.ErrUnauthorized)
		}

		return http.HandlerFunc(fn)
	}
}

func (a *tokenAuth) FromQuery(name string) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			token := r.URL.Query().Get(name)
			if token == "" {
				next.ServeHTTP(w, r)
				return
			}

			i, err := a.afn(token)
			if err != nil {
				response.MustJSONError(r.Context(), w, fmt.Errorf("%w: %s", application.ErrUnauthorized, err))
				return
			}

			next.ServeHTTP(w, r.WithContext(identity.ContextWithIdentity(r.Context(), i)))
		}

		return http.HandlerFunc(fn)
	}
}

func (a *tokenAuth) FromCookie(name string) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie(name)
			if err != nil {
				next.ServeHTTP(w, r)
				return
			}

			i, err := a.afn(cookie.Value)
			if err != nil {
				response.MustJSONError(r.Context(), w, fmt.Errorf("%w: %s", application.ErrUnauthorized, err))
				return
			}

			next.ServeHTTP(w, r.WithContext(identity.ContextWithIdentity(r.Context(), i)))
		}

		return http.HandlerFunc(fn)
	}
}

// NewToken returns new token authenticator
func NewToken(afn TokenAuthFunc) TokenAuthenticator {
	return &tokenAuth{
		afn: afn,
	}
}
