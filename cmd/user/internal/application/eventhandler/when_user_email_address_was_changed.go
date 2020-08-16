package eventhandler

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/vardius/go-api-boilerplate/cmd/user/internal/application/config"
	"github.com/vardius/go-api-boilerplate/cmd/user/internal/domain/user"
	"github.com/vardius/go-api-boilerplate/cmd/user/internal/infrastructure/persistence"
	"github.com/vardius/go-api-boilerplate/pkg/domain"
	"github.com/vardius/go-api-boilerplate/pkg/eventbus"
	"github.com/vardius/go-api-boilerplate/pkg/log"
)

// WhenUserEmailAddressWasChanged handles event
func WhenUserEmailAddressWasChanged(db *sql.DB, repository persistence.UserRepository) eventbus.EventHandler {
	fn := func(parentCtx context.Context, event domain.Event) {
		ctx, cancel := context.WithTimeout(parentCtx, time.Second*120)
		defer cancel()

		logger := log.New(config.Env.App.Environment)
		logger.Info(ctx, "[EventHandler] %s\n", event.Payload)

		e := user.EmailAddressWasChanged{}

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

		err = repository.UpdateEmail(ctx, e.ID.String(), string(e.Email))
		if err != nil {
			logger.Error(ctx, "[EventHandler] Error: %v\n", err)
			return
		}

		tx.Commit()
	}

	return fn
}
