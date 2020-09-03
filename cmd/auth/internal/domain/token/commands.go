package token

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"gopkg.in/oauth2.v4"

	"github.com/vardius/go-api-boilerplate/pkg/application"
	"github.com/vardius/go-api-boilerplate/pkg/commandbus"
	"github.com/vardius/go-api-boilerplate/pkg/domain"
	"github.com/vardius/go-api-boilerplate/pkg/errors"
	"github.com/vardius/go-api-boilerplate/pkg/executioncontext"
)

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

		token, err := repository.Get(ctx, c.ID)
		if err != nil {
			return errors.Wrap(err)
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
	TokenInfo oauth2.TokenInfo
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

		token := New()
		if err := token.Create(ctx, id, c.TokenInfo); err != nil {
			return errors.Wrap(fmt.Errorf("%w: Error when creating new token: %s", application.ErrInternal, err))
		}

		if err := repository.Save(executioncontext.WithFlag(ctx, executioncontext.LIVE), token); err != nil {
			return errors.Wrap(err)
		}

		return nil
	}

	return fn
}
