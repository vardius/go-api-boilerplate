package mysql

import (
	"context"
	"encoding/json"
	"fmt"
	"uuid"

	"github.com/vardius/go-api-boilerplate/internal/platform/es"
	"github.com/vardius/go-api-boilerplate/internal/platform/uow"
)

// EventStore is an es.Store in the events table.
type EventStore struct {
	db *DB
}

var _ es.Store = (*EventStore)(nil)

// NewEventStore returns an event store on db.
func NewEventStore(db *DB) *EventStore {
	return &EventStore{db: db}
}

// Append implements es.Store. It also rejects records whose stream or
// version don't follow from streamID and expectedVersion, since a gap in a
// stream's versions would never be noticed.
func (s *EventStore) Append(ctx context.Context, tx uow.Tx, streamID uuid.UUID, expectedVersion int, records []es.Record) error {
	t, err := SQLTx(tx)
	if err != nil {
		return err
	}
	for i, rec := range records {
		if rec.StreamID != streamID || rec.Version != expectedVersion+i+1 {
			return fmt.Errorf("mysql: append to stream %s at version %d: record %d is for stream %s at version %d",
				streamID, expectedVersion, i, rec.StreamID, rec.Version)
		}
	}

	// A plain read sees the transaction's snapshot, which may be older than
	// the stream's head. That's safe: an older head reports a conflict the
	// caller retries, and when another writer already took the next version
	// the unique key on (stream_id, version) rejects the insert below.
	var head int
	err = t.QueryRowContext(ctx, "SELECT COALESCE(MAX(version), 0) FROM events WHERE stream_id = ?", streamID[:]).Scan(&head)
	if err != nil {
		return fmt.Errorf("mysql: read the version of stream %s: %w", streamID, err)
	}
	if head != expectedVersion {
		return es.ErrConcurrency
	}
	if len(records) == 0 {
		return nil
	}

	args := make([]any, 0, 8*len(records))
	for _, rec := range records {
		meta, err := json.Marshal(rec.Meta)
		if err != nil {
			return fmt.Errorf("mysql: encode the meta of event %s: %w", rec.ID, err)
		}
		// JSON columns take text; bytes would arrive as a binary string,
		// which MySQL refuses to read as JSON.
		args = append(args, rec.ID[:], streamID[:], rec.StreamType, rec.Version, rec.Type, rec.OccurredAt, string(rec.Payload), string(meta))
	}
	query := "INSERT INTO events (id, stream_id, stream_type, version, type, occurred_at, payload, meta) VALUES " +
		valueGroups(len(records), "(?, ?, ?, ?, ?, ?, ?, ?)")
	if _, err := t.ExecContext(ctx, query, args...); err != nil {
		// The versions are checked to follow on, and event IDs are new, so
		// a duplicate key means a concurrent writer took these versions.
		if IsDuplicate(err) {
			return es.ErrConcurrency
		}
		return fmt.Errorf("mysql: append to stream %s: %w", streamID, err)
	}
	return nil
}

// Load implements es.Store.
func (s *EventStore) Load(ctx context.Context, streamID uuid.UUID) ([]es.Record, error) {
	rows, err := s.db.QueryContext(ctx,
		"SELECT id, stream_type, version, type, occurred_at, payload, meta FROM events WHERE stream_id = ? ORDER BY version",
		streamID[:])
	if err != nil {
		return nil, fmt.Errorf("mysql: load stream %s: %w", streamID, err)
	}
	defer rows.Close()

	var records []es.Record
	for rows.Next() {
		rec := es.Record{StreamID: streamID}
		var meta []byte
		if err := rows.Scan(uuidColumn{&rec.ID}, &rec.StreamType, &rec.Version, &rec.Type, &rec.OccurredAt, &rec.Payload, &meta); err != nil {
			return nil, fmt.Errorf("mysql: load stream %s: %w", streamID, err)
		}
		if err := json.Unmarshal(meta, &rec.Meta); err != nil {
			return nil, fmt.Errorf("mysql: load stream %s version %d: decode meta: %w", streamID, rec.Version, err)
		}
		records = append(records, rec)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("mysql: load stream %s: %w", streamID, err)
	}
	return records, nil
}
