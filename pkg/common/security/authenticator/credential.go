package authenticator

import (
	"net/http"

	"github.com/vardius/go-api-boilerplate/pkg/common/http/response"
	"github.com/vardius/go-api-boilerplate/pkg/common/security/identity"
)

// CredentialsAuthFunc returns Identity from username and password combination
type CredentialsAuthFunc func(username, password string) (*identity.Identity, error)

// CredentialsAuthenticator authorize by the username and password
// and adds Identity to request's Context
type CredentialsAuthenticator interface {
	// FromBasicAuth authorize by the username and password provided in the request's
	// Authorization header, if the request uses HTTP Basic Authentication.
	FromBasicAuth(next http.Handler) http.Handler
}

type credentialsAuth struct {
	afn CredentialsAuthFunc
}

func (a *credentialsAuth) FromBasicAuth(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		username, password, hasAuth := r.BasicAuth()
		if hasAuth {
			i, err := a.afn(username, password)
			if err != nil {
				w.Header().Set("WWW-Authenticate", `Basic`)

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

		next.ServeHTTP(w, r)
	}

	return http.HandlerFunc(fn)
}

// WithCredentials returns new credentials authenticator
func WithCredentials(afn CredentialsAuthFunc) CredentialsAuthenticator {
	return &credentialsAuth{
		afn: afn,
	}
}
