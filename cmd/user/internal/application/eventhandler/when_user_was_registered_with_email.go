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

// WhenUserWasRegisteredWithEmail handles event
func WhenUserWasRegisteredWithEmail(db *sql.DB, repository persistence.UserRepository) eventbus.EventHandler {
	fn := func(ctx context.Context, event domain.Event) {
		// this goroutine runs independently to request's goroutine,
		// there for recover middlewears will not recover from panic to prevent crash
		defer recoverEventHandler()

		log.Printf("[EventHandler] %s\n", event.Payload)

		e := user.WasRegisteredWithEmail{}

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

		err = repository.Add(ctx, userWasRegisteredWithEmailModel{e})
		if err != nil {
			log.Printf("[EventHandler] Error: %v\n", err)
			return
		}

		tx.Commit()
	}

	return fn
}

type userWasRegisteredWithEmailModel struct {
	e user.WasRegisteredWithEmail
}

// GetID the id
func (u userWasRegisteredWithEmailModel) GetID() string {
	return u.e.ID.String()
}

// Get full name
func (u userWasRegisteredWithEmailModel) GetName() string {
	return u.e.Name
}

// GetEmail the email
func (u userWasRegisteredWithEmailModel) GetEmail() string {
	return u.e.Email
}

// Get password
func (u userWasRegisteredWithEmailModel) GetPassword() string {
	return u.e.Password
}

// GetFacebookID facebook id
func (u userWasRegisteredWithEmailModel) GetFacebookID() string {
	return ""
}

// GetGoogleID google id
func (u userWasRegisteredWithEmailModel) GetGoogleID() string {
	return ""
}
