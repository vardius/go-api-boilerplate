package eventhandler

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/vardius/go-api-boilerplate/cmd/auth/internal/application/config"
	"github.com/vardius/go-api-boilerplate/cmd/auth/internal/domain/client"
	"github.com/vardius/go-api-boilerplate/cmd/auth/internal/infrastructure/persistence"
	"github.com/vardius/go-api-boilerplate/pkg/domain"
	"github.com/vardius/go-api-boilerplate/pkg/errors"
	"github.com/vardius/go-api-boilerplate/pkg/eventbus"
	"github.com/vardius/go-api-boilerplate/pkg/log"
)

// WhenClientWasCreated handles event
func WhenClientWasCreated(db *sql.DB, repository persistence.ClientRepository) eventbus.EventHandler {
	fn := func(parentCtx context.Context, event domain.Event) {
		ctx, cancel := context.WithTimeout(parentCtx, time.Second*120)
		defer cancel()

		logger := log.New(config.Env.App.Environment)
		logger.Info(ctx, "[EventHandler] %s\n", event.Payload)

		e := client.WasCreated{}

		err := json.Unmarshal(event.Payload, &e)
		if err != nil {
			logger.Error(ctx, "[EventHandler] Error: %v\n", errors.Wrap(err))
			return
		}

		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			logger.Error(ctx, "[EventHandler] Error: %v\n", errors.Wrap(err))
			return
		}
		defer tx.Rollback()

		if err := repository.Add(ctx, clientModel{e}); err != nil {
			logger.Error(ctx, "[EventHandler] Error: %v\n", errors.Wrap(err))
			return
		}

		if err := tx.Commit(); err != nil {
			logger.Error(ctx, "[EventHandler] Error: %v\n", errors.Wrap(err))
			return
		}
	}

	return fn
}

type clientModel struct {
	e client.WasCreated
}

// GetID client id
func (c clientModel) GetID() string {
	return c.e.ID.String()
}

// GetSecret client domain
func (c clientModel) GetSecret() string {
	return c.e.Secret
}

// GetDomain client domain
func (c clientModel) GetDomain() string {
	return c.e.Domain
}

// GetUserID user id
func (c clientModel) GetUserID() string {
	return c.e.UserID.String()
}

// GetData client data
func (c clientModel) GetData() json.RawMessage {
	return c.e.Data
}
