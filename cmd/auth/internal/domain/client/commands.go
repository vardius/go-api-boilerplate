package client

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"runtime/debug"

	"github.com/google/uuid"
	"github.com/vardius/go-api-boilerplate/internal/commandbus"
	"github.com/vardius/go-api-boilerplate/internal/errors"
	"github.com/vardius/go-api-boilerplate/internal/executioncontext"
	oauth2 "gopkg.in/oauth2.v3"
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
func OnRemove(repository Repository, db *sql.DB) commandbus.CommandHandler {
	fn := func(ctx context.Context, c Remove, out chan<- error) {
		// this goroutine runs independently to request's goroutine,
		// there for recover middlewears will not recover from panic to prevent crash
		defer recoverCommandHandler(out)

		client := repository.Get(c.ID)
		err := client.Remove()
		if err != nil {
			out <- errors.Wrap(err, errors.INTERNAL, "Error when removing client")
			return
		}

		out <- repository.Save(executioncontext.WithFlag(context.Background(), executioncontext.LIVE), client)
	}

	return commandbus.CommandHandler(fn)
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
func OnCreate(repository Repository, db *sql.DB) commandbus.CommandHandler {
	fn := func(ctx context.Context, c Create, out chan<- error) {
		// this goroutine runs independently to request's goroutine,
		// there for recover middlewears will not recover from panic to prevent crash
		defer recoverCommandHandler(out)

		client := New()
		err := client.Create(c.ClientInfo)
		if err != nil {
			out <- errors.Wrap(err, errors.INTERNAL, "Error when creating new client")
			return
		}

		out <- repository.Save(executioncontext.WithFlag(context.Background(), executioncontext.LIVE), client)
	}

	return commandbus.CommandHandler(fn)
}

func recoverCommandHandler(out chan<- error) {
	if r := recover(); r != nil {
		out <- errors.Newf(errors.INTERNAL, "[CommandHandler] Recovered in %v", r)

		// Log the Go stack trace for this panic'd goroutine.
		log.Printf("%s", debug.Stack())
	}
}
