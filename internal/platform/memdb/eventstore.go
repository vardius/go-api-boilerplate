package memdb

import (
	"context"
	"slices"
	"uuid"

	"github.com/vardius/go-api-boilerplate/internal/platform/es"
	"github.com/vardius/go-api-boilerplate/internal/platform/uow"
)

// EventStore is an es.Store in memory.
type EventStore struct {
	db      *DB
	streams map[uuid.UUID][]es.Record
}

var _ es.Store = (*EventStore)(nil)

// NewEventStore returns an empty event store on db.
func NewEventStore(db *DB) *EventStore {
	return &EventStore{db: db, streams: map[uuid.UUID][]es.Record{}}
}

// Append implements es.Store.
func (s *EventStore) Append(_ context.Context, tx uow.Tx, streamID uuid.UUID, expectedVersion int, records []es.Record) error {
	t, err := From(tx)
	if err != nil {
		return err
	}
	stream := s.streams[streamID]
	if version(stream) != expectedVersion {
		return es.ErrConcurrency
	}
	for _, rec := range records {
		rec.Payload = slices.Clone(rec.Payload)
		stream = append(stream, rec)
	}
	s.streams[streamID] = stream

	t.OnRollback(func() {
		if kept := s.streams[streamID][:len(s.streams[streamID])-len(records)]; len(kept) > 0 {
			s.streams[streamID] = kept
		} else {
			delete(s.streams, streamID)
		}
	})
	return nil
}

// Load implements es.Store.
func (s *EventStore) Load(_ context.Context, streamID uuid.UUID) ([]es.Record, error) {
	var records []es.Record
	err := s.db.Read(func() error {
		for _, rec := range s.streams[streamID] {
			rec.Payload = slices.Clone(rec.Payload)
			records = append(records, rec)
		}
		return nil
	})
	return records, err
}

func version(stream []es.Record) int {
	if len(stream) == 0 {
		return 0
	}
	return stream[len(stream)-1].Version
}
