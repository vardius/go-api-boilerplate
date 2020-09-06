package eventhandler

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/vardius/go-api-boilerplate/cmd/auth/proto"
	"github.com/vardius/go-api-boilerplate/cmd/user/internal/application/config"
	"github.com/vardius/go-api-boilerplate/cmd/user/internal/domain/user"
	"github.com/vardius/go-api-boilerplate/cmd/user/internal/infrastructure/persistence"
	"github.com/vardius/go-api-boilerplate/pkg/auth/oauth2"
	"github.com/vardius/go-api-boilerplate/pkg/domain"
	"github.com/vardius/go-api-boilerplate/pkg/errors"
	"github.com/vardius/go-api-boilerplate/pkg/eventbus"
)

// WhenUserWasRegisteredWithFacebook handles event
func WhenUserWasRegisteredWithFacebook(db *sql.DB, repository persistence.UserRepository, authClient proto.AuthenticationServiceClient) eventbus.EventHandler {
	fn := func(parentCtx context.Context, event domain.Event) error {
		ctx, cancel := context.WithTimeout(parentCtx, time.Second*120)
		defer cancel()

		e := user.WasRegisteredWithFacebook{}

		if err := json.Unmarshal(event.Payload, &e); err != nil {
			return errors.Wrap(err)
		}

		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return errors.Wrap(err)
		}
		defer tx.Rollback()

		if err := repository.Add(ctx, e); err != nil {
			return errors.Wrap(err)
		}

		if err := tx.Commit(); err != nil {
			return errors.Wrap(err)
		}

		if _, err := authClient.CreateClient(ctx, &proto.CreateClientRequest{
			UserID: e.ID.String(),
			Domain: config.Env.App.Domain,
			Scopes: oauth2.AllScopes,
		}); err != nil {
			return errors.Wrap(err)
		}

		return nil
	}

	return fn
}
