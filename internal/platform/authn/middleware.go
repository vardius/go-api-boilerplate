package authn

import (
	"net/http"
	"strings"

	"github.com/vardius/go-api-boilerplate/internal/platform/apperr"
	"github.com/vardius/go-api-boilerplate/internal/platform/httpx"
)

// AccessCookie is the cookie the auth service keeps the web app's access
// token in.
const AccessCookie = "access_token"

// Authenticate verifies the request's access token with v and puts the
// Identity in the request context. The token comes from an
// "Authorization: Bearer" header or else the access_token cookie, never the
// query string, where it would end up in logs and Referer headers. A
// request without one continues anonymous; one with a bad token gets a 401.
// An Authorization header of another scheme, such as a client's Basic
// credentials, is left for its handler.
func Authenticate(v TokenVerifier) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token, ok := accessToken(r)
			if !ok {
				next.ServeHTTP(w, r)
				return
			}
			id, err := v.Verify(r.Context(), token)
			if err != nil {
				httpx.WriteError(w, r, err)
				return
			}
			httpx.ServeWithContext(NewContext(r.Context(), id), next, w, r)
		})
	}
}

func accessToken(r *http.Request) (string, bool) {
	if scheme, token, _ := strings.Cut(r.Header.Get("Authorization"), " "); strings.EqualFold(scheme, "Bearer") {
		return strings.TrimSpace(token), true
	}
	if c, err := r.Cookie(AccessCookie); err == nil {
		return c.Value, true
	}
	return "", false
}

// Require answers 401 to an anonymous request and 403 to one whose token
// lacks any of scopes.
func Require(scopes ...string) func(http.Handler) http.Handler {
	return require(func(id Identity) error { return MissingScope(id, scopes) })
}

// RequireUser is Require that also answers 403 to a client acting for
// itself rather than for a user.
func RequireUser(scopes ...string) func(http.Handler) http.Handler {
	return require(func(id Identity) error {
		if !id.IsUser() {
			return apperr.New(apperr.ErrForbidden, "This needs a user's access token.")
		}
		return MissingScope(id, scopes)
	})
}

// RequireFirstParty answers 401 to an anonymous request and 403 unless the
// token is a user's session token from the web app, so that OAuth clients
// can't reach what only the user may do, such as managing sessions.
func RequireFirstParty() func(http.Handler) http.Handler {
	return require(func(id Identity) error {
		if !id.IsUser() || !id.IsFirstParty() {
			return apperr.New(apperr.ErrForbidden, "This needs a signed-in session.")
		}
		return nil
	})
}

func require(check func(Identity) error) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id, ok := FromContext(r.Context())
			if !ok {
				httpx.WriteError(w, r, apperr.New(apperr.ErrUnauthorized, "This needs an access token."))
				return
			}
			if err := check(id); err != nil {
				httpx.WriteError(w, r, err)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// MissingScope returns an apperr.ErrForbidden naming the first of scopes id
// lacks, or nil when it has them all.
func MissingScope(id Identity, scopes []string) error {
	for _, s := range scopes {
		if !id.Has(s) {
			return apperr.New(apperr.ErrForbidden, "This needs the "+s+" scope.")
		}
	}
	return nil
}
