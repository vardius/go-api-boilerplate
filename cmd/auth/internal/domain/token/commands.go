package token

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"

	"github.com/vardius/go-api-boilerplate/pkg/application"
	"github.com/vardius/go-api-boilerplate/pkg/commandbus"
	"github.com/vardius/go-api-boilerplate/pkg/domain"
	apperrors "github.com/vardius/go-api-boilerplate/pkg/errors"
	"github.com/vardius/go-api-boilerplate/pkg/executioncontext"
	"github.com/vardius/go-api-boilerplate/pkg/identity"
)

const (
	// RemoveAuthToken command bus contract
	RemoveAuthToken = "remove-auth-token"
)

// NewCommandFromPayload builds command by contract from json payload
func NewCommandFromPayload(contract string, payload []byte) (domain.Command, error) {
	switch contract {
	case RemoveAuthToken:
		command := Remove{}
		if err := json.Unmarshal(payload, &command); err != nil {
			return command, apperrors.Wrap(err)
		}

		return command, nil
	default:
		return nil, apperrors.Wrap(fmt.Errorf("invalid command contract: %s", contract))
	}
}

// Remove command
type Remove struct {
	ID uuid.UUID
}

// GetName returns command name
func (c Remove) GetName() string {
	return fmt.Sprintf("%T", c)
}

// OnRemove creates command handler
func OnRemove(repository Repository) commandbus.CommandHandler {
	fn := func(ctx context.Context, command domain.Command) error {
		c, ok := command.(Remove)
		if !ok {
			return apperrors.New("invalid command")
		}

		i, hasIdentity := identity.FromContext(ctx)
		if !hasIdentity {
			return apperrors.Wrap(application.ErrUnauthorized)
		}

		token, err := repository.Get(ctx, c.ID)
		if err != nil {
			return apperrors.Wrap(err)
		}
		if i.UserID.String() != token.userID.String() {
			return apperrors.Wrap(application.ErrForbidden)
		}

		if err := token.Remove(ctx); err != nil {
			return apperrors.Wrap(fmt.Errorf("%w: Error when removing token: %s", application.ErrInternal, err))
		}

		if err := repository.Save(executioncontext.WithFlag(ctx, executioncontext.LIVE), token); err != nil {
			return apperrors.Wrap(err)
		}

		return nil
	}

	return fn
}
