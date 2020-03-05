package eventhandler

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"

	"github.com/vardius/go-api-boilerplate/cmd/user/internal/domain/user"
	"github.com/vardius/go-api-boilerplate/cmd/user/internal/infrastructure/persistence"
	"github.com/vardius/go-api-boilerplate/internal/domain"
	"github.com/vardius/go-api-boilerplate/internal/eventbus"
)

// WhenUserWasRegisteredWithFacebook handles event
func WhenUserWasRegisteredWithFacebook(db *sql.DB, repository persistence.UserRepository) eventbus.EventHandler {
	fn := func(ctx context.Context, event domain.Event) {
		// this goroutine runs independently to request's goroutine,
		// there for recover middlewears will not recover from panic to prevent crash
		defer recoverEventHandler()

		log.Printf("[EventHandler] %s\n", event.Payload)

		e := user.WasRegisteredWithFacebook{}

		err := json.Unmarshal(event.Payload, &e)
		if err != nil {
			log.Printf("[EventHandler] Error: %v\n", err)
			return
		}

		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			log.Printf("[EventHandler] Error: %v\n", err)
			return
		}
		defer tx.Rollback()

		err = repository.Add(ctx, userWasRegisteredWithFacebookModel{e})
		if err != nil {
			log.Printf("[EventHandler] Error: %v\n", err)
			return
		}

		tx.Commit()
	}

	return fn
}

type userWasRegisteredWithFacebookModel struct {
	e user.WasRegisteredWithFacebook
}

// GetID the id
func (u userWasRegisteredWithFacebookModel) GetID() string {
	return u.e.ID.String()
}

// GetName the full name
func (u userWasRegisteredWithFacebookModel) GetName() string {
	return u.e.Name
}

// GetEmail the email
func (u userWasRegisteredWithFacebookModel) GetEmail() string {
	return u.e.Email
}

// GetPassword the password
func (u userWasRegisteredWithFacebookModel) GetPassword() string {
	return ""
}

// GetFacebookID facebook id
func (u userWasRegisteredWithFacebookModel) GetFacebookID() string {
	return u.e.FacebookID
}

// GetGoogleID google id
func (u userWasRegisteredWithFacebookModel) GetGoogleID() string {
	return ""
}
