package eventstore

import (
	"context"
	"sort"
	"sync"

	"github.com/google/uuid"

	"github.com/vardius/go-api-boilerplate/pkg/domain"
	baseeventstore "github.com/vardius/go-api-boilerplate/pkg/eventstore"
)

type eventStore struct {
	sync.RWMutex
	events map[string]*domain.Event
}

func (s *eventStore) Store(ctx context.Context, events []*domain.Event) error {
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

func (s *eventStore) Get(ctx context.Context, id uuid.UUID) (*domain.Event, error) {
	s.RLock()
	defer s.RUnlock()
	if val, ok := s.events[id.String()]; ok {
		return val, nil
	}

	return nil, baseeventstore.ErrEventNotFound
}

func (s *eventStore) FindAll(ctx context.Context) ([]*domain.Event, error) {
	s.RLock()
	defer s.RUnlock()
	es := make([]*domain.Event, 0, len(s.events))
	for _, val := range s.events {
		es = append(es, val)
	}
	sort.SliceStable(es, func(i, j int) bool {
		return es[i].OccurredAt.Before(es[j].OccurredAt)
	})
	return es, nil
}

func (s *eventStore) GetStream(ctx context.Context, streamID uuid.UUID, streamName string) ([]*domain.Event, error) {
	s.RLock()
	defer s.RUnlock()
	e := make([]*domain.Event, 0, 0)
	for _, val := range s.events {
		if val.StreamName == streamName && val.StreamID == streamID {
			e = append(e, val)
		}
	}
	sort.SliceStable(e, func(i, j int) bool {
		return e[i].OccurredAt.Before(e[j].OccurredAt)
	})
	return e, nil
}

func (s *eventStore) GetStreamEventsByType(ctx context.Context, streamID uuid.UUID, streamName, eventType string) ([]*domain.Event, error) {
	s.RLock()
	defer s.RUnlock()
	e := make([]*domain.Event, 0, 0)
	for _, val := range s.events {
		if val.StreamName == streamName && val.StreamID == streamID && val.Type == eventType {
			e = append(e, val)
		}
	}
	sort.SliceStable(e, func(i, j int) bool {
		return e[i].OccurredAt.Before(e[j].OccurredAt)
	})
	return e, nil
}

// New creates in memory event store
func New() baseeventstore.EventStore {
	return &eventStore{
		events: make(map[string]*domain.Event),
	}
}
