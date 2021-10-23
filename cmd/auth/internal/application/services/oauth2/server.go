package oauth2

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/vardius/go-api-boilerplate/cmd/auth/internal/application/config"
	"github.com/vardius/go-api-boilerplate/cmd/auth/internal/infrastructure/persistence"
	apperrors "github.com/vardius/go-api-boilerplate/pkg/errors"
	"github.com/vardius/go-api-boilerplate/pkg/identity"
	"github.com/vardius/go-api-boilerplate/pkg/logger"
	"gopkg.in/oauth2.v4"
	oauth2errors "gopkg.in/oauth2.v4/errors"
	oauth2server "gopkg.in/oauth2.v4/server"
)

var (
	srv        *oauth2server.Server
	onceServer sync.Once
)

// InitServer initialize the oauth2 server instance
func InitServer(
	cfg *config.Config,
	manager oauth2.Manager,
	clientRepository persistence.ClientRepository,
	timeout time.Duration,
) *oauth2server.Server {
	onceServer.Do(func() {
		srv = oauth2server.NewDefaultServer(manager)

		srv.SetAllowedGrantType(
			oauth2.Refreshing,
			oauth2.ClientCredentials, // Example usage of client credentials
			// https://github.com/go-oauth2/oauth2/blob/b46cf9f1db6551beb549ad1afe69826b3b2f1abf/example/client/client.go#L112-L128
			oauth2.AuthorizationCode, // Example usage of authorization code
			// https://github.com/go-oauth2/oauth2/blob/b46cf9f1db6551beb549ad1afe69826b3b2f1abf/example/client/client.go#L35-L62
		)
		srv.SetClientInfoHandler(oauth2server.ClientBasicHandler)

		srv.SetUserAuthorizationHandler(func(w http.ResponseWriter, r *http.Request) (userID string, err error) {
			i, isAuthorized := identity.FromContext(r.Context())
			if !isAuthorized {
				if r.Form == nil {
					if err := r.ParseForm(); err != nil {
						return "", apperrors.Wrap(err)
					}
				}

				http.Redirect(w, r, fmt.Sprintf("%s?%s", cfg.App.AuthorizeURL, r.Form.Encode()), http.StatusFound)

				return "", nil
			}

			return i.UserID.String(), nil
		})
		srv.SetClientScopeHandler(func(clientID, scope string) (allowed bool, err error) {
			ctx, cancel := context.WithTimeout(context.Background(), timeout)
			defer cancel()

			client, err := clientRepository.Get(ctx, clientID)
			if err != nil {
				return false, apperrors.Wrap(err)
			}

			tokenScopes := strings.Split(scope, ",")
			clientScopes := client.GetScopes()

			if len(tokenScopes) > len(clientScopes) {
				logger.Debug(ctx, fmt.Sprintf("Token not allowed: scopes do not match len(tokenScopes) > len(clientScopes) %v > %v", tokenScopes, clientScopes))
				return false, nil
			}

			if len(tokenScopes) == 0 || len(tokenScopes) == 0 {
				logger.Debug(ctx, fmt.Sprintf("Token not allowed: empty scopes len(tokenScopes) == 0 || len(tokenScopes) == 0 %v %v", tokenScopes, clientScopes))
				return false, nil
			}

			clientScopesMap := make(map[string]struct{}, len(clientScopes))
			for _, clientScope := range client.GetScopes() {
				clientScopesMap[clientScope] = struct{}{}
			}

			for _, s := range strings.Split(scope, " ") {
				if _, ok := clientScopesMap[s]; !ok {
					return false, nil
				}
			}

			return true, nil
		})

		srv.SetInternalErrorHandler(func(err error) (re *oauth2errors.Response) {
			return &oauth2errors.Response{
				Error: apperrors.Wrap(err),
			}
		})

		srv.SetResponseErrorHandler(func(re *oauth2errors.Response) {
			logger.Error(context.Background(), fmt.Sprintf("[oAuth2|Server] response error: %v", re))
		})
	})

	return srv
}
