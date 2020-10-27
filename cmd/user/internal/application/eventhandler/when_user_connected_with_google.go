package eventhandler

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/vardius/go-api-boilerplate/cmd/user/internal/domain/user"
	"github.com/vardius/go-api-boilerplate/cmd/user/internal/infrastructure/persistence"
	"github.com/vardius/go-api-boilerplate/pkg/domain"
	apperrors "github.com/vardius/go-api-boilerplate/pkg/errors"
	"github.com/vardius/go-api-boilerplate/pkg/eventbus"
)

// WhenUserConnectedWithGoogle handles event
func WhenUserConnectedWithGoogle(db *sql.DB, repository persistence.UserRepository) eventbus.EventHandler {
	fn := func(parentCtx context.Context, event domain.Event) error {
		ctx, cancel := context.WithTimeout(parentCtx, time.Second*120)
		defer cancel()

		var e user.ConnectedWithGoogle
		if err := json.Unmarshal(event.Payload, &e); err != nil {
			return apperrors.Wrap(err)
		}

		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return apperrors.Wrap(err)
		}
		defer tx.Rollback()

		if err := repository.UpdateGoogleID(ctx, e.ID.String(), e.GoogleID); err != nil {
			return apperrors.Wrap(err)
		}

		if err := tx.Commit(); err != nil {
			return apperrors.Wrap(err)
		}

		return nil
	}

	return fn
}
