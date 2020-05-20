package token

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"runtime/debug"

	"github.com/google/uuid"
	"gopkg.in/oauth2.v3"

	"github.com/vardius/go-api-boilerplate/pkg/application"
	"github.com/vardius/go-api-boilerplate/pkg/commandbus"
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
func OnRemove(repository Repository, db *sql.DB) commandbus.CommandHandler {
	fn := func(ctx context.Context, c Remove, out chan<- error) {
		// this goroutine runs independently to request's goroutine,
		// therefor recover middlewears will not recover from panic to prevent crash
		defer recoverCommandHandler(out)

		token := repository.Get(c.ID)
		err := token.Remove()
		if err != nil {
			out <- errors.Wrap(fmt.Errorf("%w: Error when removing token: %s", application.ErrInternal, err))
			return
		}

		out <- repository.Save(executioncontext.WithFlag(ctx, executioncontext.LIVE), token)
	}

	return commandbus.CommandHandler(fn)
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
func OnCreate(repository Repository, db *sql.DB) commandbus.CommandHandler {
	fn := func(ctx context.Context, c Create, out chan<- error) {
		// this goroutine runs independently to request's goroutine,
		// therefor recover middlewears will not recover from panic to prevent crash
		defer recoverCommandHandler(out)

		id, err := uuid.NewRandom()
		if err != nil {
			out <- errors.Wrap(fmt.Errorf("%w: Could not generate new id: %s", application.ErrInternal, err))
			return
		}

		token := New()
		err = token.Create(id, c.TokenInfo)
		if err != nil {
			out <- errors.Wrap(fmt.Errorf("%w: Error when creating new token: %s", application.ErrInternal, err))
			return
		}

		out <- repository.Save(executioncontext.WithFlag(ctx, executioncontext.LIVE), token)
	}

	return commandbus.CommandHandler(fn)
}

func recoverCommandHandler(out chan<- error) {
	if r := recover(); r != nil {
		out <- errors.Wrap(fmt.Errorf("[CommandHandler] Recovered in %v", r))

		// Log the Go stack trace for this panic'd goroutine.
		log.Printf("%s\n", debug.Stack())
	}
}
