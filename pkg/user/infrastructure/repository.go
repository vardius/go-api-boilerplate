package infrastructure

import (
	"context"

	"github.com/google/uuid"
	"github.com/vardius/go-api-boilerplate/pkg/common/infrastructure/eventbus"
	"github.com/vardius/go-api-boilerplate/pkg/common/infrastructure/eventstore"
	"github.com/vardius/go-api-boilerplate/pkg/user/domain/user"
)

type eventSourcedRepository struct {
	streamName string
	eventStore eventstore.EventStore
	eventBus   eventbus.EventBus
}

// Save current user changes to event store and publish each event with an event bus
func (r *eventSourcedRepository) Save(ctx context.Context, u *user.User) error {
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
func (r *eventSourcedRepository) Get(id uuid.UUID) *user.User {
	events := r.eventStore.GetStream(id, r.streamName)

	u := user.New()
	u.FromHistory(events)

	return u
}

// NewUserEventSourcedRepository creates new user event sourced repository
func NewUserEventSourcedRepository(streamName string, store eventstore.EventStore, bus eventbus.EventBus) user.EventSourcedRepository {
	return &eventSourcedRepository{streamName, store, bus}
}
