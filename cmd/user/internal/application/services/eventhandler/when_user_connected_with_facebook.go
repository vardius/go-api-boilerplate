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

// WhenUserConnectedWithFacebook handles event
func WhenUserConnectedWithFacebook(db *sql.DB, repository persistence.UserRepository) eventbus.EventHandler {
	fn := func(parentCtx context.Context, event domain.Event) error {
		ctx, cancel := context.WithTimeout(parentCtx, time.Second*120)
		defer cancel()

		e := user.ConnectedWithFacebook{}
		if err := json.Unmarshal(event.Payload, &e); err != nil {
			return apperrors.Wrap(err)
		}

		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return apperrors.Wrap(err)
		}
		defer tx.Rollback()

		if err := repository.UpdateFacebookID(ctx, e.ID.String(), e.FacebookID); err != nil {
			return apperrors.Wrap(err)
		}

		if err := tx.Commit(); err != nil {
			return apperrors.Wrap(err)
		}

		return nil
	}

	return fn
}
