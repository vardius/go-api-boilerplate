/*
Package repository holds event sourced repositories
*/
package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/vardius/go-api-boilerplate/pkg/common/infrastructure/eventbus"
	"github.com/vardius/go-api-boilerplate/pkg/common/infrastructure/eventstore"
	"github.com/vardius/go-api-boilerplate/pkg/user/domain/user"
)

type userRepository struct {
	eventStore eventstore.EventStore
	eventBus   eventbus.EventBus
}

// Save current user changes to event store and publish each event with an event bus
func (r *userRepository) Save(ctx context.Context, u *user.User) error {
	err := r.eventStore.Store(u.Changes())
	if err != nil {
		return err
	}

	for _, event := range u.Changes() {
		r.eventBus.Publish(ctx, event.Metadata.Type, *event)
	}

	return nil
}

// Get user with current state applied
func (r *userRepository) Get(id uuid.UUID) *user.User {
	events := r.eventStore.GetStream(id, user.StreamName)

	u := user.New()
	u.FromHistory(events)

	return u
}

// NewUser creates new user event sourced repository
func NewUser(store eventstore.EventStore, bus eventbus.EventBus) user.Repository {
	return &userRepository{store, bus}
}
