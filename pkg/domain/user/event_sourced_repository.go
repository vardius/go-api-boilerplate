package user

import (
	"app/pkg/domain"
	"context"

	"github.com/google/uuid"
)

type eventSourcedRepository struct {
	streamName string
	eventStore domain.EventStore
	eventBus   domain.EventBus
}

// Save current aggregate root changes to event store and publish each event with event bus
func (r *eventSourcedRepository) Save(ctx context.Context, u *User) error {
	r.eventStore.Store(u.Changes())

	for _, event := range u.Changes() {
		r.eventBus.Publish(event.Metadata.Type, ctx, event)
	}

	return nil
}

// Get aggregate root with current state applied
func (r *eventSourcedRepository) Get(id uuid.UUID) *User {
	events := r.eventStore.GetStream(id, r.streamName)

	aggregateRoot := New()
	aggregateRoot.FromHistory(events)

	return aggregateRoot
}

func newEventSourcedRepository(streamName string, store domain.EventStore, bus domain.EventBus) *eventSourcedRepository {
	return &eventSourcedRepository{streamName, store, bus}
}
