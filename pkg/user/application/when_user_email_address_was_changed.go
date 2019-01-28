package application

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"

	"github.com/vardius/go-api-boilerplate/pkg/common/domain"
	"github.com/vardius/go-api-boilerplate/pkg/common/infrastructure/eventbus"
	"github.com/vardius/go-api-boilerplate/pkg/user/domain/user"
	"github.com/vardius/go-api-boilerplate/pkg/user/infrastructure/persistence"
)

// WhenUserEmailAddressWasChanged handles event
func WhenUserEmailAddressWasChanged(db *sql.DB, repository persistence.UserRepository) eventbus.EventHandler {
	fn := func(ctx context.Context, event domain.Event) {
		log.Printf("[EventHandler] %s", event.Payload)

		e := &user.EmailAddressWasChanged{}

		err := json.Unmarshal(event.Payload, e)
		if err != nil {
			log.Printf("[EventHandler] Error: %v", err)
		}

		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			log.Printf("[EventHandler] Error: %v", err)
		}
		defer tx.Rollback()

		err = repository.UpdateEmail(ctx, e.ID.String(), e.Email)
		if err != nil {
			log.Printf("[EventHandler] Error: %v", err)
			return
		}

		tx.Commit()
	}

	return eventbus.EventHandler(fn)
}
