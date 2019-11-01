package oauth2

import (
	"context"
	"database/sql"
	"sync"

	"github.com/vardius/go-api-boilerplate/internal/errors"
	"github.com/vardius/golog"
	oauth2 "gopkg.in/oauth2.v3"
	oauth2_errors "gopkg.in/oauth2.v3/errors"
	oauth2_server "gopkg.in/oauth2.v3/server"
)

var (
	gServer    *oauth2_server.Server
	onceServer sync.Once
)

// InitServer initialize the oauth2 server instance
func InitServer(manager oauth2.Manager, db *sql.DB, logger golog.Logger, secretKey string) *oauth2_server.Server {
	onceServer.Do(func() {
		gServer = oauth2_server.NewDefaultServer(manager)

		gServer.SetAllowedGrantType(oauth2.PasswordCredentials, oauth2.Refreshing)
		gServer.SetClientInfoHandler(oauth2_server.ClientFormHandler)

		gServer.SetPasswordAuthorizationHandler(func(email, password string) (string, error) {
			// we allow password grant only within our system, due to email passwordless authentication
			// password value here should contain secretKey
			if password != secretKey {
				return "", errors.New(errors.UNAUTHORIZED, "Invalid client, user password does not match secret key")
			}

			userID, err := getUserIDByEmail(context.Background(), db, email)
			if err != nil {
				return "", errors.Wrapf(err, errors.UNAUTHORIZED, "Could not find user id for given email (%s)", email)
			}

			return userID, nil
		})

		gServer.SetInternalErrorHandler(func(err error) (re *oauth2_errors.Response) {
			logger.Error(context.Background(), "oAuth2 Server internal error: %s\n", err.Error())

			return &oauth2_errors.Response{
				Error: errors.Wrap(err, errors.INTERNAL, "Internal error"),
			}
		})

		gServer.SetResponseErrorHandler(func(re *oauth2_errors.Response) {
			logger.Error(context.Background(), "oAuth2 Server response error: %s\n%v\n", re.Error.Error(), re)
		})
	})

	return gServer
}

func getUserIDByEmail(ctx context.Context, db *sql.DB, email string) (id string, err error) {
	row := db.QueryRowContext(ctx, `SELECT id FROM users WHERE emailAddress=?`, email)
	e := row.Scan(&id)

	switch {
	case e == sql.ErrNoRows:
		err = errors.Wrap(err, errors.NOTFOUND, "User ID not found")
	case e != nil:
		err = errors.Wrap(err, errors.INTERNAL, "Error while scanning users table")
	}

	return
}
