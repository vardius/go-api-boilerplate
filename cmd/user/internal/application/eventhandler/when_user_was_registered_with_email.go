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

// GetProvider the name
func (u userWasRegisteredWithEmailModel) GetProvider() string {
	return u.e.Provider
}

// GetName the name
func (u userWasRegisteredWithEmailModel) GetName() string {
	return u.e.Name
}

// GetEmail the email
func (u userWasRegisteredWithEmailModel) GetEmail() string {
	return u.e.Email
}

// GetPassword the password
func (u userWasRegisteredWithEmailModel) GetPassword() string {
	return u.e.Password
}

// GetNickName the nickname
func (u userWasRegisteredWithEmailModel) GetNickName() string {
	return u.e.NickName
}

// GetLocation the location
func (u userWasRegisteredWithEmailModel) GetLocation() string {
	return u.e.Location
}

// GetAvatarURL the avatarurl
func (u userWasRegisteredWithEmailModel) GetAvatarURL() string {
	return u.e.AvatarURL
}

// GetDescription the description
func (u userWasRegisteredWithEmailModel) GetDescription() string {
	return u.e.Description
}

// GetUserID the userid
func (u userWasRegisteredWithEmailModel) GetUserID() string {
	return u.e.UserID
}

// GetRefreshToken the refreshtoken
func (u userWasRegisteredWithEmailModel) GetRefreshToken() string {
	return u.e.RefreshToken
}
