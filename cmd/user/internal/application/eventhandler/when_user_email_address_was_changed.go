package eventhandler

import (
	"context"
	"time"

	"github.com/vardius/go-api-boilerplate/cmd/user/internal/domain/user"
	"github.com/vardius/go-api-boilerplate/cmd/user/internal/infrastructure/persistence"
	"github.com/vardius/go-api-boilerplate/pkg/domain"
	apperrors "github.com/vardius/go-api-boilerplate/pkg/errors"
	"github.com/vardius/go-api-boilerplate/pkg/eventbus"
)

// WhenUserEmailAddressWasChanged handles event
func WhenUserEmailAddressWasChanged(repository persistence.UserRepository) eventbus.EventHandler {
	fn := func(parentCtx context.Context, event *domain.Event) error {
		ctx, cancel := context.WithTimeout(parentCtx, time.Second*120)
		defer cancel()

		e := event.Payload.(*user.EmailAddressWasChanged)

		if err := repository.UpdateEmail(ctx, e.ID.String(), string(e.Email)); err != nil {
			return apperrors.Wrap(err)
		}

		return nil
	}

	return fn
}
