package user

import (
	"context"
	"github.com/vardius/go-api-boilerplate/pkg/domain"

	"github.com/google/uuid"
)

type eventSourcedRepository struct {
	streamName string
	eventStore domain.EventStore
	eventBus   domain.EventBus
}

// Save current user changes to event store and publish each event with an event bus
func (r *eventSourcedRepository) Save(ctx context.Context, u *User) error {
	r.eventStore.Store(u.Changes())

	for _, event := range u.Changes() {
		r.eventBus.Publish(ctx, event.Metadata.Type, *event)
	}

	return nil
}

// Get user with current state applied
func (r *eventSourcedRepository) Get(id uuid.UUID) *User {
	events := r.eventStore.GetStream(id, r.streamName)

	user := New()
	user.FromHistory(events)

	return user
}

func newRepository(streamName string, store domain.EventStore, bus domain.EventBus) *eventSourcedRepository {
	return &eventSourcedRepository{streamName, store, bus}
}
