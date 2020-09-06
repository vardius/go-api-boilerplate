/*
Package eventstore provides mysql implementation of domain event store
*/
package eventstore

import (
	"context"
	"database/sql"
	systemErrors "errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/vardius/go-api-boilerplate/pkg/domain"
	"github.com/vardius/go-api-boilerplate/pkg/errors"
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

	query := "INSERT INTO events (event_id, event_type, stream_id, stream_name, stream_version, occurred_at, payload) VALUES "
	values := make([]interface{}, 0, lenEvents*7)

	if lenEvents > 1 {
		for i := 0; i < lenEvents-1; i++ {
			query += "(?, ?, ?, ?, ?, ?, ?),"
			values = append(values,
				events[i].ID.String(),
				events[i].Type,
				events[i].StreamID.String(),
				events[i].StreamName,
				events[i].StreamVersion,
				events[i].OccurredAt.UTC(),
				events[i].Payload,
			)
		}
	}

	i := lenEvents - 1
	query += "(?, ?, ?, ?, ?, ?, ?)"
	values = append(values,
		events[i].ID.String(),
		events[i].Type,
		events[i].StreamID.String(),
		events[i].StreamName,
		events[i].StreamVersion,
		events[i].OccurredAt.UTC(),
		events[i].Payload,
	)

	stmt, err := s.db.PrepareContext(ctx, query)
	if err != nil {
		return errors.Wrap(err)
	}
	defer stmt.Close()

	if _, err := stmt.ExecContext(ctx, values...); err != nil {
		return errors.Wrap(err)
	}

	return nil
}

func (s *eventStore) Get(ctx context.Context, id uuid.UUID) (domain.Event, error) {
	row := s.db.QueryRowContext(ctx, `SELECT event_id, event_type, stream_id, stream_name, stream_version, occurred_at, payload FROM events WHERE event_id=? LIMIT 1`, id.String())

	event := domain.Event{}

	var (
		eventId  string
		streamID string
	)
	err := row.Scan(
		&eventId,
		&event.Type,
		&streamID,
		&event.StreamName,
		&event.StreamVersion,
		&event.OccurredAt,
		&event.Payload,
	)

	switch {
	case systemErrors.Is(err, sql.ErrNoRows):
		return event, errors.Wrap(fmt.Errorf("%w: %s", baseeventstore.ErrEventNotFound, err))
	case err != nil:
		return event, errors.Wrap(fmt.Errorf(`%w: SELECT event_id, event_type, stream_id, stream_name, stream_version, occurred_at, payload FROM events WHERE id=%s LIMIT 1`, err, id.String()))
	}

	event.ID = uuid.MustParse(eventId)
	event.StreamID = uuid.MustParse(streamID)

	return event, nil
}

func (s *eventStore) FindAll(ctx context.Context) ([]domain.Event, error) {
	return nil, errors.Wrap(fmt.Errorf("should never load all events from mysql"))
}

func (s *eventStore) GetStream(ctx context.Context, streamID uuid.UUID, streamName string) ([]domain.Event, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT event_id, event_type, stream_id, stream_name, stream_version, occurred_at, payload FROM events WHERE stream_id=? AND stream_name=? ORDER BY distinct_id ASC`, streamID.String(), streamName)
	if err != nil {
		return nil, errors.Wrap(fmt.Errorf(`%w: SELECT id, type, stream_id, stream_name, stream_version, occurred_at, payload FROM events WHERE stream_id=%s AND stream_name=%s ORDER BY distinct_id DESC`, err, streamID.String(), streamName))
	}
	defer rows.Close()

	var events []domain.Event

	for rows.Next() {
		event := domain.Event{}

		var (
			id       string
			streamID string
		)
		if err := rows.Scan(
			&id,
			&event.Type,
			&streamID,
			&event.StreamName,
			&event.StreamVersion,
			&event.OccurredAt,
			&event.Payload,
		); err != nil {
			return nil, errors.Wrap(err)
		}

		event.ID = uuid.MustParse(id)
		event.StreamID = uuid.MustParse(streamID)

		events = append(events, event)
	}

	if err := rows.Err(); err != nil {
		return nil, errors.Wrap(err)
	}

	return events, nil
}

// New creates in mysql event store
func New(db *sql.DB) baseeventstore.EventStore {
	return &eventStore{db}
}
