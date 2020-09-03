package eventhandler

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/vardius/go-api-boilerplate/cmd/auth/internal/domain/client"
	"github.com/vardius/go-api-boilerplate/cmd/auth/internal/infrastructure/persistence"
	"github.com/vardius/go-api-boilerplate/pkg/domain"
	"github.com/vardius/go-api-boilerplate/pkg/errors"
	"github.com/vardius/go-api-boilerplate/pkg/eventbus"
)

// WhenClientWasCreated handles event
func WhenClientWasCreated(db *sql.DB, repository persistence.ClientRepository) eventbus.EventHandler {
	fn := func(parentCtx context.Context, event domain.Event) error {
		ctx, cancel := context.WithTimeout(parentCtx, time.Second*120)
		defer cancel()

		e := client.WasCreated{}
		if err := json.Unmarshal(event.Payload, &e); err != nil {
			return errors.Wrap(err)
		}

		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return errors.Wrap(err)
		}
		defer tx.Rollback()

		if err := repository.Add(ctx, clientModel{e}); err != nil {
			return errors.Wrap(err)
		}

		if err := tx.Commit(); err != nil {
			return errors.Wrap(err)
		}

		return nil
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
