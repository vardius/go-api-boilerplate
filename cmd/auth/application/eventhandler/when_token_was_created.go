package eventhandler

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"

	"github.com/vardius/go-api-boilerplate/cmd/auth/domain/token"
	"github.com/vardius/go-api-boilerplate/cmd/auth/infrastructure/persistence"
	"github.com/vardius/go-api-boilerplate/pkg/domain"
	"github.com/vardius/go-api-boilerplate/pkg/eventbus"
)

// WhenTokenWasCreated handles event
func WhenTokenWasCreated(db *sql.DB, repository persistence.TokenRepository) eventbus.EventHandler {
	fn := func(ctx context.Context, event domain.Event) {
		// this goroutine runs independently to request's goroutine,
		// there for recover middlewears will not recover from panic to prevent crash
		defer recoverEventHandler()

		log.Printf("[EventHandler] %s", event.Payload)

		e := &token.WasCreated{}

		err := json.Unmarshal(event.Payload, e)
		if err != nil {
			log.Printf("[EventHandler] Error: %v", err)
			return
		}

		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			log.Printf("[EventHandler] Error: %v", err)
			return
		}
		defer tx.Rollback()

		t := &persistence.Token{
			ID:       e.ID.String(),
			UserID:   e.UserID.String(),
			ClientID: e.ClientID.String(),
			Code:     e.Code,
			Access:   e.Access,
			Refresh:  e.Refresh,
			Info:     e.Info,
		}

		err = repository.Add(ctx, t)
		if err != nil {
			log.Printf("[EventHandler] Error: %v", err)
			return
		}

		tx.Commit()
	}

	return eventbus.EventHandler(fn)
}
