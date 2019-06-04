/*
Package repository holds event sourced repositories
*/
package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/vardius/go-api-boilerplate/cmd/auth/domain/client"
	"github.com/vardius/go-api-boilerplate/pkg/eventbus"
	"github.com/vardius/go-api-boilerplate/pkg/eventstore"
)

type clientRepository struct {
	eventStore eventstore.EventStore
	eventBus   eventbus.EventBus
}

// Save current client changes to event store and publish each event with an event bus
func (r *clientRepository) Save(ctx context.Context, u *client.Client) error {
	err := r.eventStore.Store(u.Changes())
	if err != nil {
		return err
	}

	for _, event := range u.Changes() {
		r.eventBus.Publish(ctx, event.Metadata.Type, *event)
	}

	return nil
}

// Get client with current state applied
func (r *clientRepository) Get(id uuid.UUID) *client.Client {
	events := r.eventStore.GetStream(id, client.StreamName)

	u := client.New()
	u.FromHistory(events)

	return u
}

// NewClientRepository creates new client event sourced repository
func NewClientRepository(store eventstore.EventStore, bus eventbus.EventBus) client.Repository {
	return &clientRepository{store, bus}
}
