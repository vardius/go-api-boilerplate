package oauth2

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"gopkg.in/oauth2.v4"
	oauth2errors "gopkg.in/oauth2.v4/errors"
	oauth2server "gopkg.in/oauth2.v4/server"

	"github.com/vardius/go-api-boilerplate/cmd/auth/internal/application/config"
	"github.com/vardius/go-api-boilerplate/cmd/auth/internal/infrastructure/persistence"
	"github.com/vardius/go-api-boilerplate/pkg/application"
	"github.com/vardius/go-api-boilerplate/pkg/errors"
	"github.com/vardius/go-api-boilerplate/pkg/identity"
	"github.com/vardius/go-api-boilerplate/pkg/log"
)

var (
	srv        *oauth2server.Server
	onceServer sync.Once
)

// InitServer initialize the oauth2 server instance
func InitServer(
	manager oauth2.Manager,
	logger *log.Logger,
	userRepository persistence.UserRepository,
	clientRepository persistence.ClientRepository,
	secretKey string,
	timeout time.Duration,
) *oauth2server.Server {
	onceServer.Do(func() {
		srv = oauth2server.NewDefaultServer(manager)

		srv.SetAllowedGrantType(
			oauth2.PasswordCredentials,
			oauth2.Refreshing,
			oauth2.ClientCredentials, // Example usage of client credentials
			// https://github.com/go-oauth2/oauth2/blob/b46cf9f1db6551beb549ad1afe69826b3b2f1abf/example/client/client.go#L112-L128

			// @TODO: AuthorizationCode  uncomment below and look for other todos if you want to enable this flow
			// oauth2.AuthorizationCode, // Example usage of authorization code
			// https://github.com/go-oauth2/oauth2/blob/b46cf9f1db6551beb549ad1afe69826b3b2f1abf/example/client/client.go#L35-L62
		)
		srv.SetClientInfoHandler(oauth2server.ClientFormHandler)

		srv.SetPasswordAuthorizationHandler(func(email, password string) (string, error) {
			ctx, cancel := context.WithTimeout(context.Background(), timeout)
			defer cancel()

			// we allow password grant only within our system, due to email passwordless authentication
			// password value here should contain secretKey
			if password != secretKey {
				return "", errors.Wrap(fmt.Errorf("%w: Invalid client, user password does not match secret key", application.ErrUnauthorized))
			}

			user, err := userRepository.GetByEmail(ctx, email)
			if err != nil {
				return "", errors.Wrap(err)
			}

			return user.GetID(), nil
		})
		srv.SetUserAuthorizationHandler(func(w http.ResponseWriter, r *http.Request) (userID string, err error) {
			i, isAuthorized := identity.FromContext(r.Context())
			if !isAuthorized {
				http.Redirect(w, r, config.Env.App.AuthorizeURL, http.StatusFound)

				return "", nil
			}

			return i.UserID.String(), nil
		})
		srv.SetClientScopeHandler(func(clientID, scope string) (allowed bool, err error) {
			ctx, cancel := context.WithTimeout(context.Background(), timeout)
			defer cancel()

			client, err := clientRepository.Get(ctx, clientID)
			if err != nil {
				return false, errors.Wrap(err)
			}

			tokenScopes := strings.Split(scope, ",")
			clientScopes := client.GetScopes()

			if len(tokenScopes) > len(clientScopes) {
				return false, nil
			}

			if len(tokenScopes) == 0 || len(tokenScopes) == 0 {
				return false, nil
			}

			clientScopesMap := make(map[string]struct{}, len(clientScopes))
			for _, clientScope := range client.GetScopes() {
				clientScopesMap[clientScope] = struct{}{}
			}

			for _, s := range strings.Split(scope, ",") {
				if _, ok := clientScopesMap[s]; !ok {
					return false, nil
				}
			}

			return false, nil
		})

		srv.SetInternalErrorHandler(func(err error) (re *oauth2errors.Response) {
			logger.Error(context.Background(), "[oAuth2|Server] internal error: %s", err.Error())

			return &oauth2errors.Response{
				Error: errors.Wrap(err),
			}
		})

		srv.SetResponseErrorHandler(func(re *oauth2errors.Response) {
			logger.Error(context.Background(), "[oAuth2|Server] response error: %v %v", re.Error, re)
		})
	})

	return srv
}
