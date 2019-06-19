package eventhandler

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"

	"github.com/vardius/go-api-boilerplate/cmd/auth/domain/token"
	"github.com/vardius/go-api-boilerplate/cmd/auth/infrastructure/persistence"
	auth_mysql "github.com/vardius/go-api-boilerplate/cmd/auth/infrastructure/persistence/mysql"
	"github.com/vardius/go-api-boilerplate/pkg/domain"
	"github.com/vardius/go-api-boilerplate/pkg/eventbus"
	"github.com/vardius/go-api-boilerplate/pkg/mysql"
)

// WhenTokenWasCreated handles event
func WhenTokenWasCreated(db *sql.DB, repository persistence.TokenRepository) eventbus.EventHandler {
	fn := func(ctx context.Context, event domain.Event) {
		// this goroutine runs independently to request's goroutine,
		// there for recover middlewears will not recover from panic to prevent crash
		defer recoverEventHandler()

		log.Printf("[EventHandler] %s", event.Payload)

		e := token.WasCreated{}

		err := json.Unmarshal(event.Payload, &e)
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

		err = repository.Add(ctx, auth_mysql.Token{
			ID:       e.ID.String(),
			ClientID: e.ClientID.String(),
			UserID:   e.UserID.String(),
			Scope:    e.Scope,
			Access:   e.Access,
			Refresh:  e.Refresh,
			Code: mysql.NullString{sql.NullString{
				String: e.Code,
				Valid:  e.Code != "",
			}},
			Data: e.Data,
		})
		if err != nil {
			log.Printf("[EventHandler] Error: %v", err)
			return
		}

		tx.Commit()
	}

	return eventbus.EventHandler(fn)
}
