package oauth2

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	"gopkg.in/oauth2.v4"
	oauth2errors "gopkg.in/oauth2.v4/errors"
	oauth2server "gopkg.in/oauth2.v4/server"

	"github.com/vardius/go-api-boilerplate/pkg/application"
	"github.com/vardius/go-api-boilerplate/pkg/errors"
	"github.com/vardius/go-api-boilerplate/pkg/log"
)

var (
	gServer    *oauth2server.Server
	onceServer sync.Once
)

// InitServer initialize the oauth2 server instance
func InitServer(manager oauth2.Manager, db *sql.DB, logger *log.Logger, secretKey string, timeout time.Duration) *oauth2server.Server {
	onceServer.Do(func() {
		gServer = oauth2server.NewDefaultServer(manager)

		gServer.SetAllowedGrantType(oauth2.PasswordCredentials, oauth2.Refreshing)
		gServer.SetClientInfoHandler(oauth2server.ClientFormHandler)

		gServer.SetPasswordAuthorizationHandler(func(email, password string) (string, error) {
			ctx, cancel := context.WithTimeout(context.Background(), timeout)
			defer cancel()

			// we allow password grant only within our system, due to email passwordless authentication
			// password value here should contain secretKey
			if password != secretKey {
				return "", errors.Wrap(fmt.Errorf("%w: Invalid client, user password does not match secret key", application.ErrUnauthorized))
			}

			userID, err := getUserIDByEmail(ctx, db, email)
			if err != nil {
				return "", errors.Wrap(fmt.Errorf("%w: Could not find user id for given email (%s): %s", application.ErrUnauthorized, email, err))
			}

			return userID, nil
		})

		gServer.SetInternalErrorHandler(func(err error) (re *oauth2errors.Response) {
			logger.Error(context.Background(), "[oAuth2|Server] internal error: %s\n", err.Error())

			return &oauth2errors.Response{
				Error: errors.Wrap(err),
			}
		})

		gServer.SetResponseErrorHandler(func(re *oauth2errors.Response) {
			logger.Error(context.Background(), "[oAuth2|Server] response error: %s\n%v\n", re.Error.Error(), re)
		})
	})

	return gServer
}

func getUserIDByEmail(ctx context.Context, db *sql.DB, email string) (id string, err error) {
	row := db.QueryRowContext(ctx, `SELECT id FROM users WHERE email_address=? LIMIT 1`, email)
	e := row.Scan(&id)

	switch {
	case e == sql.ErrNoRows:
		err = errors.Wrap(fmt.Errorf("%w: User UserID not found: %s", application.ErrNotFound, e))
	case e != nil:
		err = errors.Wrap(fmt.Errorf("error while scanning users table: %w", e))
	}

	return
}
