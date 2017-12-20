package authenticator

import (
	"encoding/base64"
	"errors"
	"net/http"
	"strings"

	"github.com/vardius/go-api-boilerplate/pkg/http/response"
	"github.com/vardius/go-api-boilerplate/pkg/security/identity"
)

// TokenAuthFunc returns Identity from token
type TokenAuthFunc func(token string) (*identity.Identity, error)

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
						response.WithError(r.Context(), response.HTTPError{
							Code:    http.StatusUnauthorized,
							Error:   err,
							Message: http.StatusText(http.StatusUnauthorized),
						})
						return
					}

					next.ServeHTTP(w, r.WithContext(identity.ContextWithIdentity(r.Context(), i)))
					return
				}
			}

			w.Header().Set("WWW-Authenticate", `Bearer realm="`+realm+`"`)
			response.WithError(r.Context(), response.HTTPError{
				Code:    http.StatusUnauthorized,
				Error:   errors.New(http.StatusText(http.StatusUnauthorized)),
				Message: http.StatusText(http.StatusUnauthorized),
			})
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
				response.WithError(r.Context(), response.HTTPError{
					Code:    http.StatusUnauthorized,
					Error:   err,
					Message: http.StatusText(http.StatusUnauthorized),
				})
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
				response.WithError(r.Context(), response.HTTPError{
					Code:    http.StatusUnauthorized,
					Error:   err,
					Message: http.StatusText(http.StatusUnauthorized),
				})
				return
			}

			next.ServeHTTP(w, r.WithContext(identity.ContextWithIdentity(r.Context(), i)))
		}

		return http.HandlerFunc(fn)
	}
}

// WithToken returns new token authenticator
func WithToken(afn TokenAuthFunc) TokenAuthenticator {
	return &tokenAuth{
		afn: afn,
	}
}
