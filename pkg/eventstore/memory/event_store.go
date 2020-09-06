/*
Package eventstore provides memory implementation of domain event store
*/
package eventstore

import (
	"context"
	"sync"

	"github.com/google/uuid"

	"github.com/vardius/go-api-boilerplate/pkg/domain"
	baseeventstore "github.com/vardius/go-api-boilerplate/pkg/eventstore"
)

type eventStore struct {
	sync.RWMutex
	events map[string]domain.Event
}

func (s *eventStore) Store(ctx context.Context, events []domain.Event) error {
	if len(events) == 0 {
		return nil
	}

	s.Lock()
	defer s.Unlock()

	// @TODO: check event version
	for _, e := range events {
		s.events[e.ID.String()] = e
	}

	return nil
}

func (s *eventStore) Get(ctx context.Context, id uuid.UUID) (domain.Event, error) {
	s.RLock()
	defer s.RUnlock()
	if val, ok := s.events[id.String()]; ok {
		return val, nil
	}

	return domain.NullEvent, baseeventstore.ErrEventNotFound
}

func (s *eventStore) FindAll(ctx context.Context) ([]domain.Event, error) {
	s.RLock()
	defer s.RUnlock()
	es := make([]domain.Event, 0, len(s.events))
	for _, val := range s.events {
		es = append(es, val)
	}
	return es, nil
}

func (s *eventStore) GetStream(ctx context.Context, streamID uuid.UUID, streamName string) ([]domain.Event, error) {
	s.RLock()
	defer s.RUnlock()
	e := make([]domain.Event, 0, 0)
	for _, val := range s.events {
		if val.StreamName == streamName && val.StreamID == streamID {
			e = append(e, val)
		}
	}
	return e, nil
}

// New creates in memory event store
func New() baseeventstore.EventStore {
	return &eventStore{
		events: make(map[string]domain.Event),
	}
}
