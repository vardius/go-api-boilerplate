package eventhandler

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/vardius/go-api-boilerplate/cmd/user/internal/application/config"
	"github.com/vardius/go-api-boilerplate/cmd/user/internal/domain/user"
	"github.com/vardius/go-api-boilerplate/cmd/user/internal/infrastructure/persistence"
	"github.com/vardius/go-api-boilerplate/pkg/domain"
	"github.com/vardius/go-api-boilerplate/pkg/eventbus"
	"github.com/vardius/go-api-boilerplate/pkg/log"
)

// WhenUserWasRegisteredWithGoogle handles event
func WhenUserWasRegisteredWithGoogle(db *sql.DB, repository persistence.UserRepository) eventbus.EventHandler {
	fn := func(parentCtx context.Context, event domain.Event) {
		ctx, cancel := context.WithTimeout(parentCtx, time.Second*120)
		defer cancel()

		logger := log.New(config.Env.App.Environment)
		logger.Info(ctx, "[EventHandler] %s\n", event.Payload)

		e := user.WasRegisteredWithGoogle{}

		err := json.Unmarshal(event.Payload, &e)
		if err != nil {
			logger.Error(ctx, "[EventHandler] Error: %v\n", err)
			return
		}

		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			logger.Error(ctx, "[EventHandler] Error: %v\n", err)
			return
		}
		defer tx.Rollback()

		err = repository.Add(ctx, userWasRegisteredWithGoogleModel{e})
		if err != nil {
			logger.Error(ctx, "[EventHandler] Error: %v\n", err)
			return
		}

		tx.Commit()
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
