package client

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"gopkg.in/oauth2.v4"

	"github.com/vardius/go-api-boilerplate/pkg/commandbus"
	"github.com/vardius/go-api-boilerplate/pkg/domain"
	"github.com/vardius/go-api-boilerplate/pkg/errors"
	"github.com/vardius/go-api-boilerplate/pkg/executioncontext"
)

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

		client, err := repository.Get(ctx, c.ID)
		if err != nil {
			return errors.Wrap(err)
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
	ClientInfo oauth2.ClientInfo
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

		client := New()
		if err := client.Create(ctx, c.ClientInfo); err != nil {
			return errors.Wrap(err)
		}

		// we block here until event handler is done
		// this is because when other services request access token after creating client
		// we want handler to be finished and client persisted in storage
		if err := repository.SaveAndAcknowledge(executioncontext.WithFlag(ctx, executioncontext.LIVE), client); err != nil {
			return errors.Wrap(err)
		}

		return nil
	}

	return fn
}
