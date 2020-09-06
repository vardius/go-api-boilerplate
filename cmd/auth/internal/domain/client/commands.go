package client

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
	// CreateClientCredentials command bus contract
	CreateClientCredentials = "create-client-credentials"
	// RemoveClientCredentials command bus contract
	RemoveClientCredentials = "remove-client-credentials"
)

// NewCommandFromPayload builds command by contract from json payload
func NewCommandFromPayload(contract string, payload []byte) (domain.Command, error) {
	switch contract {
	case CreateClientCredentials:
		command := Create{}
		err := unmarshalPayload(payload, &command)

		return command, errors.Wrap(err)
	case RemoveClientCredentials:
		command := Remove{}
		err := unmarshalPayload(payload, &command)

		return command, errors.Wrap(err)
	default:
		return nil, errors.New("Invalid command contract")
	}
}

// Remove command
type Remove struct {
	ID uuid.UUID `json:"id"`
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

		client, err := repository.Get(ctx, c.ID)
		if err != nil {
			return errors.Wrap(err)
		}
		if i.UserID.String() != client.userID.String() {
			return errors.Wrap(application.ErrForbidden)
		}

		if err := client.Remove(ctx); err != nil {
			return errors.Wrap(err)
		}

		if err := repository.Save(executioncontext.WithFlag(ctx, executioncontext.LIVE), client); err != nil {
			return errors.Wrap(err)
		}

		return nil
	}

	return fn
}

// Create command
type Create struct {
	UserID      uuid.UUID `json:"user_id"`
	Domain      string    `json:"domain"`
	RedirectURL string    `json:"redirect_url"`
	Scopes      []string  `json:"scopes"`
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

		id, err := uuid.NewRandom()
		if err != nil {
			return errors.Wrap(fmt.Errorf("%w: Could not generate new id: %s", application.ErrInternal, err))
		}
		secret, err := uuid.NewRandom()
		if err != nil {
			return errors.Wrap(fmt.Errorf("%w: Could not generate new secret: %s", application.ErrInternal, err))
		}

		i, hasIdentity := identity.FromContext(ctx)
		if !hasIdentity {
			return errors.Wrap(application.ErrUnauthorized)
		}
		if i.UserID.String() != c.UserID.String() {
			return errors.Wrap(application.ErrForbidden)
		}

		client := New()
		if err := client.Create(ctx, id, secret.String(), c.UserID, c.Domain, c.RedirectURL, c.Scopes...); err != nil {
			return errors.Wrap(err)
		}

		if err := repository.Save(executioncontext.WithFlag(ctx, executioncontext.LIVE), client); err != nil {
			return errors.Wrap(err)
		}

		return nil
	}

	return fn
}
