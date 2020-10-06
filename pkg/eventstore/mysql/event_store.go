/*
Package eventstore provides mysql implementation of domain event store
*/
package eventstore

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/vardius/go-api-boilerplate/pkg/domain"
	apperrors "github.com/vardius/go-api-boilerplate/pkg/errors"
	baseeventstore "github.com/vardius/go-api-boilerplate/pkg/eventstore"
)

type eventStore struct {
	db *sql.DB
}

func (s *eventStore) Store(ctx context.Context, events []domain.Event) error {
	lenEvents := len(events)
	if lenEvents == 0 {
		return nil
	}

	query := "INSERT INTO events (event_id, event_type, stream_id, stream_name, stream_version, occurred_at, payload, metadata) VALUES "
	values := make([]interface{}, 0, lenEvents*7)

	if lenEvents > 1 {
		for i := 0; i < lenEvents-1; i++ {
			query += "(?, ?, ?, ?, ?, ?, ?, ?),"
			values = append(values,
				events[i].ID.String(),
				events[i].Type,
				events[i].StreamID.String(),
				events[i].StreamName,
				events[i].StreamVersion,
				events[i].OccurredAt.UTC(),
				events[i].Payload,
				events[i].Metadata,
			)
		}
	}

	i := lenEvents - 1
	query += "(?, ?, ?, ?, ?, ?, ?, ?)"
	values = append(values,
		events[i].ID.String(),
		events[i].Type,
		events[i].StreamID.String(),
		events[i].StreamName,
		events[i].StreamVersion,
		events[i].OccurredAt.UTC(),
		events[i].Payload,
		events[i].Metadata,
	)

	stmt, err := s.db.PrepareContext(ctx, query)
	if err != nil {
		return apperrors.Wrap(err)
	}
	defer stmt.Close()

	if _, err := stmt.ExecContext(ctx, values...); err != nil {
		return apperrors.Wrap(err)
	}

	return nil
}

func (s *eventStore) Get(ctx context.Context, id uuid.UUID) (domain.Event, error) {
	query := `SELECT event_id, event_type, stream_id, stream_name, stream_version, occurred_at, payload, metadata FROM events WHERE event_id=? LIMIT 1`
	row := s.db.QueryRowContext(ctx, query, id.String())

	var event domain.Event

	var (
		eventId  string
		streamID string
		metadata sql.NullString
	)
	err := row.Scan(
		&eventId,
		&event.Type,
		&streamID,
		&event.StreamName,
		&event.StreamVersion,
		&event.OccurredAt,
		&event.Payload,
		&metadata,
	)

	switch {
	case errors.Is(err, sql.ErrNoRows):
		return event, apperrors.Wrap(fmt.Errorf("%w: %s", baseeventstore.ErrEventNotFound, err))
	case err != nil:
		return event, apperrors.Wrap(fmt.Errorf("%w: %s (%s)", err, query, id.String()))
	}

	event.ID = uuid.MustParse(eventId)
	event.StreamID = uuid.MustParse(streamID)
	event.Metadata = json.RawMessage(metadata.String)

	return event, nil
}

func (s *eventStore) FindAll(ctx context.Context) ([]domain.Event, error) {
	return nil, apperrors.Wrap(fmt.Errorf("should never load all events from mysql"))
}

func (s *eventStore) GetStream(ctx context.Context, streamID uuid.UUID, streamName string) ([]domain.Event, error) {
	query := `SELECT event_id, event_type, stream_id, stream_name, stream_version, occurred_at, payload, metadata FROM events WHERE stream_id=? AND stream_name=? ORDER BY distinct_id ASC`
	rows, err := s.db.QueryContext(ctx, query, streamID.String(), streamName)
	if err != nil {
		return nil, apperrors.Wrap(fmt.Errorf("%w: %s (%s, %s)", err, query, streamID.String(), streamName))
	}
	defer rows.Close()

	var events []domain.Event

	for rows.Next() {
		var (
			event    domain.Event
			id       string
			streamID string
			metadata sql.NullString
		)
		if err := rows.Scan(
			&id,
			&event.Type,
			&streamID,
			&event.StreamName,
			&event.StreamVersion,
			&event.OccurredAt,
			&event.Payload,
			&metadata,
		); err != nil {
			return nil, apperrors.Wrap(err)
		}

		event.ID = uuid.MustParse(id)
		event.StreamID = uuid.MustParse(streamID)
		event.Metadata = json.RawMessage(metadata.String)

		events = append(events, event)
	}

	if err := rows.Err(); err != nil {
		return nil, apperrors.Wrap(err)
	}

	return events, nil
}

// New creates in mysql event store
func New(db *sql.DB) baseeventstore.EventStore {
	return &eventStore{db}
}
