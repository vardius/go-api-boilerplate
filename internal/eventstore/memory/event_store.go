/*
Package eventstore provides memory implementation of domain event store
*/
package eventstore

import (
	"sync"

	"github.com/google/uuid"

	"github.com/vardius/go-api-boilerplate/internal/domain"
	"github.com/vardius/go-api-boilerplate/internal/errors"
	baseeventstore "github.com/vardius/go-api-boilerplate/internal/eventstore"
)

type eventStore struct {
	sync.RWMutex
	events map[string]domain.Event
}

func (s *eventStore) Store(events []domain.Event) error {
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

func (s *eventStore) Get(id uuid.UUID) (domain.Event, error) {
	s.RLock()
	defer s.RUnlock()
	if val, ok := s.events[id.String()]; ok {
		return val, nil
	}
	return domain.NullEvent, errors.Wrap(ErrEventNotFound, errors.NOTFOUND, "Not found any items")
}

func (s *eventStore) FindAll() []domain.Event {
	s.RLock()
	defer s.RUnlock()
	es := make([]domain.Event, 0, len(s.events))
	for _, val := range s.events {
		es = append(es, val)
	}
	return es
}

func (s *eventStore) GetStream(streamID uuid.UUID, streamName string) []domain.Event {
	s.RLock()
	defer s.RUnlock()
	e := make([]domain.Event, 0, 0)
	for _, val := range s.events {
		if val.Metadata.StreamName == streamName && val.Metadata.StreamID == streamID {
			e = append(e, val)
		}
	}
	return e
}

// New creates in memory event store
func New() baseeventstore.EventStore {
	return &eventStore{
		events: make(map[string]domain.Event),
	}
}
