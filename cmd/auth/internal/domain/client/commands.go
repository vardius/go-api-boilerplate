package client

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"

	"github.com/vardius/go-api-boilerplate/pkg/commandbus"
	"github.com/vardius/go-api-boilerplate/pkg/domain"
	apperrors "github.com/vardius/go-api-boilerplate/pkg/errors"
	"github.com/vardius/go-api-boilerplate/pkg/executioncontext"
	"github.com/vardius/go-api-boilerplate/pkg/identity"
)

const (
	// CreateClientCredentials command bus contract
	CreateClientCredentials = "client-create-credentials"
	// RemoveClientCredentials command bus contract
	RemoveClientCredentials = "client-remove-credentials"
)

var (
	RemoveName = (Remove{}).GetName()
	CreateName = (Create{}).GetName()
)

// NewCommandFromPayload builds command by contract from json payload
func NewCommandFromPayload(contract string, payload []byte) (domain.Command, error) {
	switch contract {
	case CreateClientCredentials:
		var command Create
		if err := json.Unmarshal(payload, &command); err != nil {
			return command, apperrors.Wrap(err)
		}
		return command, nil
	case RemoveClientCredentials:
		var command Remove
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
			return apperrors.New("invalid command")
		}

		i, hasIdentity := identity.FromContext(ctx)
		if !hasIdentity {
			return apperrors.Wrap(apperrors.ErrUnauthorized)
		}

		client, err := repository.Get(ctx, c.ID)
		if err != nil {
			return apperrors.Wrap(err)
		}
		if i.UserID.String() != client.userID.String() {
			return apperrors.Wrap(apperrors.ErrForbidden)
		}

		if err := client.Remove(ctx); err != nil {
			return apperrors.Wrap(err)
		}

		if err := repository.Save(executioncontext.WithFlag(ctx, executioncontext.LIVE), client); err != nil {
			return apperrors.Wrap(err)
		}

		return nil
	}

	return fn
}

// Create command
type Create struct {
	Domain      string   `json:"domain"`
	RedirectURL string   `json:"redirect_url"`
	Scopes      []string `json:"scopes"`
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
			return apperrors.New("invalid command")
		}

		id, err := uuid.NewRandom()
		if err != nil {
			return apperrors.Wrap(fmt.Errorf("%w: Could not generate new id: %s", apperrors.ErrInternal, err))
		}
		secret, err := uuid.NewRandom()
		if err != nil {
			return apperrors.Wrap(fmt.Errorf("%w: Could not generate new secret: %s", apperrors.ErrInternal, err))
		}

		i, hasIdentity := identity.FromContext(ctx)
		if !hasIdentity {
			return apperrors.Wrap(apperrors.ErrUnauthorized)
		}

		client := New()
		if err := client.Create(ctx, id, secret, i.UserID, c.Domain, c.RedirectURL, c.Scopes...); err != nil {
			return apperrors.Wrap(err)
		}

		if err := repository.Save(executioncontext.WithFlag(ctx, executioncontext.LIVE), client); err != nil {
			return apperrors.Wrap(err)
		}

		return nil
	}

	return fn
}
