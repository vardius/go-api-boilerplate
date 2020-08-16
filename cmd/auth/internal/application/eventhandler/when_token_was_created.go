package eventhandler

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/vardius/go-api-boilerplate/cmd/auth/internal/application/config"
	"github.com/vardius/go-api-boilerplate/cmd/auth/internal/domain/token"
	"github.com/vardius/go-api-boilerplate/cmd/auth/internal/infrastructure/persistence"
	"github.com/vardius/go-api-boilerplate/pkg/domain"
	"github.com/vardius/go-api-boilerplate/pkg/eventbus"
	"github.com/vardius/go-api-boilerplate/pkg/log"
)

// WhenTokenWasCreated handles event
func WhenTokenWasCreated(db *sql.DB, repository persistence.TokenRepository) eventbus.EventHandler {
	fn := func(parentCtx context.Context, event domain.Event) {
		ctx, cancel := context.WithTimeout(parentCtx, time.Second*120)
		defer cancel()

		logger := log.New(config.Env.App.Environment)
		logger.Info(ctx, "[EventHandler] %s\n", event.Payload)

		e := token.WasCreated{}

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

		err = repository.Add(ctx, tokenModel{e})
		if err != nil {
			logger.Error(ctx, "[EventHandler] Error: %v\n", err)
			return
		}

		tx.Commit()
	}

	return fn
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
