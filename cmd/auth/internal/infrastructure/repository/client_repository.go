/*
Package repository holds event sourced repositories
*/
package repository

import (
	"context"

	"github.com/google/uuid"

	"github.com/vardius/go-api-boilerplate/cmd/auth/internal/domain/client"
	"github.com/vardius/go-api-boilerplate/pkg/application"
	"github.com/vardius/go-api-boilerplate/pkg/errors"
	"github.com/vardius/go-api-boilerplate/pkg/eventbus"
	"github.com/vardius/go-api-boilerplate/pkg/eventstore"
)

type clientRepository struct {
	eventStore eventstore.EventStore
	eventBus   eventbus.EventBus
}

// Save current client changes to event store and publish each event with an event bus
func (r *clientRepository) Save(ctx context.Context, u client.Client) error {
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

// Get client with current state applied
func (r *clientRepository) Get(id uuid.UUID) (client.Client, error) {
	events := r.eventStore.GetStream(id, client.StreamName)

	if len(events) == 0 {
		return client.Client{}, application.ErrNotFound
	}

	return client.FromHistory(events), nil
}

// NewClientRepository creates new client event sourced repository
func NewClientRepository(store eventstore.EventStore, bus eventbus.EventBus) client.Repository {
	return &clientRepository{store, bus}
}
