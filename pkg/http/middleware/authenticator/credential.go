package authenticator

import (
	"fmt"
	"net/http"

	"github.com/vardius/golog"

	apperrors "github.com/vardius/go-api-boilerplate/pkg/errors"
	"github.com/vardius/go-api-boilerplate/pkg/identity"
)

// CredentialsAuthFunc returns Identity from username and password combination
type CredentialsAuthFunc func(username, password string) (identity.Identity, error)

// CredentialsAuthenticator authorize by the username and password
// and adds Identity to request's Context
type CredentialsAuthenticator interface {
	// FromBasicAuth authorize by the username and password provided in the request's
	// Authorization header, if the request uses HTTP Basic Authentication.
	FromBasicAuth(name string, logger golog.Logger) func(next http.Handler) http.Handler
}

type credentialsAuth struct {
	afn CredentialsAuthFunc
}

func (a *credentialsAuth) FromBasicAuth(realm string, logger golog.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			username, password, hasAuth := r.BasicAuth()
			if hasAuth {
				i, err := a.afn(username, password)
				if err != nil {
					w.Header().Set("WWW-Authenticate", fmt.Sprintf(`Basic realm="%s"`, realm))

					logger.Warning(r.Context(), "[HTTP] failed to authenticate from basic auth: %v", apperrors.Wrap(err))
				}

				next.ServeHTTP(w, r.WithContext(identity.ContextWithIdentity(r.Context(), &i)))
				return
			}

			next.ServeHTTP(w, r)
		}

		return http.HandlerFunc(fn)
	}
}

// NewCredentials returns new credentials authenticator
func NewCredentials(afn CredentialsAuthFunc) CredentialsAuthenticator {
	return &credentialsAuth{
		afn: afn,
	}
}
