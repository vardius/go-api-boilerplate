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

		err = repository.Add(ctx, tokenModel{e})
		if err != nil {
			log.Printf("[EventHandler] Error: %v", err)
			return
		}

		tx.Commit()
	}

	return eventbus.EventHandler(fn)
}

type tokenModel struct {
	e token.WasCreated
}

// GetID the id
func (t tokenModel) GetID() string {
	return t.e.ID.String()
}

// GetClientID the client id
func (t tokenModel) GetClientID() string {
	return t.e.ClientID.String()
}

// GetUserID the user id
func (t tokenModel) GetUserID() string {
	return t.e.UserID.String()
}

// GetAccess access token
func (t tokenModel) GetAccess() string {
	return t.e.Access
}

// GetRefresh refresh token
func (t tokenModel) GetRefresh() string {
	return t.e.Refresh
}

// GetScope get scope of authorization
func (t tokenModel) GetScope() string {
	return t.e.Scope
}

// GetCode authorization code
func (t tokenModel) GetCode() string {
	return t.e.Code
}

// GetData token data
func (t tokenModel) GetData() json.RawMessage {
	return t.e.Data
}
