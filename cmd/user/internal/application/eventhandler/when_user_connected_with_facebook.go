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

// WhenUserConnectedWithFacebook handles event
func WhenUserConnectedWithFacebook(db *sql.DB, repository persistence.UserRepository) eventbus.EventHandler {
	fn := func(ctx context.Context, event domain.Event) {
		// this goroutine runs independently to request's goroutine,
		// there for recover middlewears will not recover from panic to prevent crash
		defer recoverEventHandler()

		log.Printf("[EventHandler] %s\n", event.Payload)

		e := user.ConnectedWithFacebook{}

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

		err = repository.UpdateFacebookID(ctx, e.ID.String(), e.FacebookID)
		if err != nil {
			log.Printf("[EventHandler] Error: %v\n", err)
			return
		}

		tx.Commit()
	}

	return fn
}
