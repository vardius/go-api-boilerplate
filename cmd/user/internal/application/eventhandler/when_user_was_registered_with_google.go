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
	"github.com/vardius/go-api-boilerplate/pkg/domain"
	"github.com/vardius/go-api-boilerplate/pkg/errors"
	"github.com/vardius/go-api-boilerplate/pkg/eventbus"
)

// WhenUserWasRegisteredWithGoogle handles event
func WhenUserWasRegisteredWithGoogle(db *sql.DB, repository persistence.UserRepository, authClient proto.AuthenticationServiceClient) eventbus.EventHandler {
	fn := func(parentCtx context.Context, event domain.Event) error {
		ctx, cancel := context.WithTimeout(parentCtx, time.Second*120)
		defer cancel()

		e := user.WasRegisteredWithGoogle{}

		if err := json.Unmarshal(event.Payload, &e); err != nil {
			return errors.Wrap(err)
		}

		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return errors.Wrap(err)
		}
		defer tx.Rollback()

		if err := repository.Add(ctx, userWasRegisteredWithGoogleModel{e}); err != nil {
			return errors.Wrap(err)
		}

		if err := tx.Commit(); err != nil {
			return errors.Wrap(err)
		}

		if _, err := authClient.CreateClient(ctx, &proto.CreateClientRequest{
			UserID: e.ID.String(),
			Domain: config.Env.App.Domain,
		}); err != nil {
			return errors.Wrap(err)
		}

		return nil
	}

	return fn
}

type userWasRegisteredWithGoogleModel struct {
	e user.WasRegisteredWithGoogle
}

// GetID the id
func (u userWasRegisteredWithGoogleModel) GetID() string {
	return u.e.ID.String()
}

// GetEmail the email
func (u userWasRegisteredWithGoogleModel) GetEmail() string {
	return string(u.e.Email)
}

// GetFacebookID facebook id
func (u userWasRegisteredWithGoogleModel) GetFacebookID() string {
	return ""
}

// GetGoogleID google id
func (u userWasRegisteredWithGoogleModel) GetGoogleID() string {
	return u.e.GoogleID
}
