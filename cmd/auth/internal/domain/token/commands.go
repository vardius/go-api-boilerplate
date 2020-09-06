package token

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/vardius/go-api-boilerplate/pkg/application"
	"github.com/vardius/go-api-boilerplate/pkg/commandbus"
	"github.com/vardius/go-api-boilerplate/pkg/domain"
	"github.com/vardius/go-api-boilerplate/pkg/errors"
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
		if err := unmarshalPayload(payload, &command); err != nil {
			return command, errors.Wrap(err)
		}

		return command, nil
	default:
		return nil, errors.New("Invalid command contract")
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
			return errors.New("invalid command")
		}

		i, hasIdentity := identity.FromContext(ctx)
		if !hasIdentity {
			return errors.Wrap(application.ErrUnauthorized)
		}

		token, err := repository.Get(ctx, c.ID)
		if err != nil {
			return errors.Wrap(err)
		}
		if i.UserID.String() != token.userID.String() {
			return errors.Wrap(application.ErrForbidden)
		}

		if err := token.Remove(ctx); err != nil {
			return errors.Wrap(fmt.Errorf("%w: Error when removing token: %s", application.ErrInternal, err))
		}

		if err := repository.Save(executioncontext.WithFlag(ctx, executioncontext.LIVE), token); err != nil {
			return errors.Wrap(err)
		}

		return nil
	}

	return fn
}

// Create command
type Create struct {
	ClientID uuid.UUID `json:"client_id"`
	UserID   uuid.UUID `json:"user_id"`
	Code     string    `json:"code"`
	Scope    string    `json:"scope"`
	Access   string    `json:"access"`
	Refresh  string    `json:"refresh"`
}

// GetName returns command name
func (c Create) GetName() string {
	return fmt.Sprintf("%T", c)
}

// OnCreate creates command handler
func OnCreate(repository Repository) commandbus.CommandHandler {
	fn := func(ctx context.Context, command domain.Command) error {
		c, ok := command.(Create)
		if !ok {
			return errors.New("invalid command")
		}

		i, hasIdentity := identity.FromContext(ctx)
		if !hasIdentity {
			return errors.Wrap(application.ErrUnauthorized)
		}
		if i.UserID.String() != c.UserID.String() {
			return errors.Wrap(application.ErrForbidden)
		}

		id, err := uuid.NewRandom()
		if err != nil {
			return errors.Wrap(fmt.Errorf("%w: Could not generate new id: %s", application.ErrInternal, err))
		}

		token := New()
		if err := token.Create(
			ctx,
			id,
			c.ClientID,
			c.UserID,
			c.Code,
			c.Scope,
			c.Access,
			c.Refresh,
		); err != nil {
			return errors.Wrap(fmt.Errorf("%w: Error when creating new token: %s", application.ErrInternal, err))
		}

		if err := repository.Save(executioncontext.WithFlag(ctx, executioncontext.LIVE), token); err != nil {
			return errors.Wrap(err)
		}

		return nil
	}

	return fn
}
