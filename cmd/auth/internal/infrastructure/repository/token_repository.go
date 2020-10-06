/*
Package repository holds event sourced repositories
*/
package repository

import (
	"context"

	"github.com/google/uuid"

	"github.com/vardius/go-api-boilerplate/cmd/auth/internal/domain/token"
	"github.com/vardius/go-api-boilerplate/pkg/application"
	apperrors "github.com/vardius/go-api-boilerplate/pkg/errors"
	"github.com/vardius/go-api-boilerplate/pkg/eventbus"
	"github.com/vardius/go-api-boilerplate/pkg/eventstore"
)

type tokenRepository struct {
	eventStore eventstore.EventStore
	eventBus   eventbus.EventBus
}

// Save current token changes to event store and publish each event with an event bus
func (r *tokenRepository) Save(ctx context.Context, u token.Token) error {
	if err := r.eventStore.Store(ctx, u.Changes()); err != nil {
		return apperrors.Wrap(err)
	}

	for _, event := range u.Changes() {
		if err := r.eventBus.Publish(ctx, event); err != nil {
			return apperrors.Wrap(err)
		}
	}

	return nil
}

// Save current token changes to event store and publish each event with an event bus
// blocks until event handlers are finished
func (r *tokenRepository) SaveAndAcknowledge(ctx context.Context, u token.Token) error {
	if err := r.eventStore.Store(ctx, u.Changes()); err != nil {
		return apperrors.Wrap(err)
	}

	for _, event := range u.Changes() {
		if err := r.eventBus.PublishAndAcknowledge(ctx, event); err != nil {
			return apperrors.Wrap(err)
		}
	}

	return nil
}

// Get token with current state applied
func (r *tokenRepository) Get(ctx context.Context, id uuid.UUID) (token.Token, error) {
	events, err := r.eventStore.GetStream(ctx, id, token.StreamName)
	if err != nil {
		return token.Token{}, apperrors.Wrap(err)
	}

	if len(events) == 0 {
		return token.Token{}, application.ErrNotFound
	}

	return token.FromHistory(ctx, events)
}

// NewTokenRepository creates new token event sourced repository
func NewTokenRepository(store eventstore.EventStore, bus eventbus.EventBus) token.Repository {
	return &tokenRepository{store, bus}
}
