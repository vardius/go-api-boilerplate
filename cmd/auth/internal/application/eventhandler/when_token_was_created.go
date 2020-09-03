package eventhandler

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/vardius/go-api-boilerplate/cmd/auth/internal/domain/token"
	"github.com/vardius/go-api-boilerplate/cmd/auth/internal/infrastructure/persistence"
	"github.com/vardius/go-api-boilerplate/pkg/domain"
	"github.com/vardius/go-api-boilerplate/pkg/errors"
	"github.com/vardius/go-api-boilerplate/pkg/eventbus"
)

// WhenTokenWasCreated handles event
func WhenTokenWasCreated(db *sql.DB, repository persistence.TokenRepository) eventbus.EventHandler {
	fn := func(parentCtx context.Context, event domain.Event) error {
		ctx, cancel := context.WithTimeout(parentCtx, time.Second*120)
		defer cancel()

		e := token.WasCreated{}
		if err := json.Unmarshal(event.Payload, &e); err != nil {
			return errors.Wrap(err)
		}

		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return errors.Wrap(err)
		}
		defer tx.Rollback()

		if err := repository.Add(ctx, tokenModel{e}); err != nil {
			return errors.Wrap(err)
		}

		if err := tx.Commit(); err != nil {
			return errors.Wrap(err)
		}

		return nil
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
