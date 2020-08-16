/*
Package repository holds event sourced repositories
*/
package repository

import (
	"context"

	"github.com/google/uuid"

	"github.com/vardius/go-api-boilerplate/cmd/auth/internal/domain/token"
	"github.com/vardius/go-api-boilerplate/pkg/errors"
	"github.com/vardius/go-api-boilerplate/pkg/eventbus"
	"github.com/vardius/go-api-boilerplate/pkg/eventstore"
)

type tokenRepository struct {
	eventStore eventstore.EventStore
	eventBus   eventbus.EventBus
}

// Save current token changes to event store and publish each event with an event bus
func (r *tokenRepository) Save(ctx context.Context, u token.Token) error {
	err := r.eventStore.Store(u.Changes())
	if err != nil {
		return errors.Wrap(err)
	}

	for _, event := range u.Changes() {
		if err := r.eventBus.Publish(ctx, event); err != nil {
			return errors.Wrap(err)
		}
	}

	return nil
}

// Get token with current state applied
func (r *tokenRepository) Get(id uuid.UUID) token.Token {
	events := r.eventStore.GetStream(id, token.StreamName)

	return token.FromHistory(events)
}

// NewTokenRepository creates new token event sourced repository
func NewTokenRepository(store eventstore.EventStore, bus eventbus.EventBus) token.Repository {
	return &tokenRepository{store, bus}
}
