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

const createTableSQLFormat = `
CREATE TABLE IF NOT EXISTS %s
(
    distinct_id    INT          NOT NULL AUTO_INCREMENT,
    event_id       CHAR(36)     NOT NULL,
    event_type     VARCHAR(255) NOT NULL,
    stream_id      CHAR(36)     NOT NULL,
    stream_name    VARCHAR(255) NOT NULL,
    stream_version INT          NOT NULL,
    occurred_at    DATETIME     NOT NULL,
    payload        JSON         NOT NULL,
    metadata       JSON DEFAULT NULL,
    PRIMARY KEY (distinct_id),
    UNIQUE KEY u_event_id (event_id),
    INDEX i_stream_id_stream_name_event_type (stream_id, stream_name, event_type)
)
    ENGINE = InnoDB
    DEFAULT CHARSET = utf8
    COLLATE = utf8_bin;
`

type eventStore struct {
	tableName string
	db        *sql.DB
}

// New creates in mysql event store
func New(ctx context.Context, tableName string, db *sql.DB) (baseeventstore.EventStore, error) {
	if _, err := db.ExecContext(ctx, fmt.Sprintf(createTableSQLFormat, tableName)); err != nil {
		return nil, apperrors.Wrap(err)
	}

	return &eventStore{tableName: tableName, db: db}, nil
}

func addEventToInsert(values []interface{}, event *domain.Event) ([]interface{}, error) {
	payload, err := json.Marshal(event.Payload)
	if err != nil {
		return nil, apperrors.Wrap(err)
	}
	metadata, err := json.Marshal(event.Metadata)
	if err != nil {
		return nil, apperrors.Wrap(err)
	}

	return append(values,
		event.ID.String(),
		event.Type,
		event.StreamID.String(),
		event.StreamName,
		event.StreamVersion,
		event.OccurredAt.UTC(),
		payload,
		metadata,
	), nil
}

func (s *eventStore) Store(ctx context.Context, events []*domain.Event) error {
	lenEvents := len(events)
	if lenEvents == 0 {
		return nil
	}

	query := "INSERT INTO " + s.tableName + " (event_id, event_type, stream_id, stream_name, stream_version, occurred_at, payload, metadata) VALUES "
	values := make([]interface{}, 0, lenEvents*7)

	if lenEvents > 1 {
		for i := 0; i < lenEvents-1; i++ {
			var err error
			query += "(?, ?, ?, ?, ?, ?, ?, ?),"
			values, err = addEventToInsert(values, events[i])
			if err != nil {
				return apperrors.Wrap(err)
			}
		}
	}

	i := lenEvents - 1
	var err error
	query += "(?, ?, ?, ?, ?, ?, ?, ?),"
	values, err = addEventToInsert(values, events[i])
	if err != nil {
		return apperrors.Wrap(err)
	}

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

func (s *eventStore) Get(ctx context.Context, id uuid.UUID) (*domain.Event, error) {
	query := "SELECT event_id, event_type, stream_id, stream_name, stream_version, occurred_at, payload, metadata FROM " + s.tableName + " WHERE event_id=? LIMIT 1"
	row := s.db.QueryRowContext(ctx, query, id.String())

	var (
		event    domain.Event
		eventId  string
		streamID string
		payload  json.RawMessage
		metadata sql.NullString
	)
	err := row.Scan(
		&eventId,
		&event.Type,
		&streamID,
		&event.StreamName,
		&event.StreamVersion,
		&event.OccurredAt,
		payload,
		&metadata,
	)

	switch {
	case errors.Is(err, sql.ErrNoRows):
		return nil, apperrors.Wrap(fmt.Errorf("%w: %s", baseeventstore.ErrEventNotFound, err))
	case err != nil:
		return nil, apperrors.Wrap(fmt.Errorf("%w: %s (%s)", err, query, id.String()))
	}

	event.Payload, err = getRawEvent(event.Type, payload)
	if err != nil {
		return nil, apperrors.Wrap(err)
	}
	event.Metadata, err = getEventMetadata(json.RawMessage(metadata.String))
	if err != nil {
		return nil, apperrors.Wrap(err)
	}

	event.ID = uuid.MustParse(eventId)
	event.StreamID = uuid.MustParse(streamID)

	return &event, nil
}

func (s *eventStore) FindAll(ctx context.Context) ([]*domain.Event, error) {
	return nil, apperrors.Wrap(fmt.Errorf("should never load all events from mysql"))
}

func (s *eventStore) GetStream(ctx context.Context, streamID uuid.UUID, streamName string) ([]*domain.Event, error) {
	query := "SELECT event_id, event_type, stream_name, stream_version, occurred_at, payload, metadata FROM " + s.tableName + " WHERE stream_id=? AND stream_name=? ORDER BY distinct_id ASC"
	rows, err := s.db.QueryContext(ctx, query, streamID.String(), streamName)
	if err != nil {
		return nil, apperrors.Wrap(fmt.Errorf("%w: %s (%s, %s)", err, query, streamID.String(), streamName))
	}
	defer rows.Close()

	var events []*domain.Event

	for rows.Next() {
		var (
			event    domain.Event
			id       string
			payload  json.RawMessage
			metadata sql.NullString
		)
		if err := rows.Scan(
			&id,
			&event.Type,
			&event.StreamName,
			&event.StreamVersion,
			&event.OccurredAt,
			payload,
			&metadata,
		); err != nil {
			return nil, apperrors.Wrap(err)
		}

		event.Payload, err = getRawEvent(event.Type, payload)
		if err != nil {
			return nil, apperrors.Wrap(err)
		}
		event.Metadata, err = getEventMetadata(json.RawMessage(metadata.String))
		if err != nil {
			return nil, apperrors.Wrap(err)
		}

		event.ID = uuid.MustParse(id)
		event.StreamID = streamID

		events = append(events, &event)
	}

	if err := rows.Err(); err != nil {
		return nil, apperrors.Wrap(err)
	}

	return events, nil
}

func (s *eventStore) GetStreamEventsByType(ctx context.Context, streamID uuid.UUID, streamName, eventType string) ([]*domain.Event, error) {
	query := "SELECT event_id, stream_name, stream_version, occurred_at, payload, metadata FROM " + s.tableName + " WHERE stream_id=? AND stream_name=? AND event_type=? ORDER BY distinct_id ASC"
	rows, err := s.db.QueryContext(ctx, query, streamID.String(), streamName, eventType)
	if err != nil {
		return nil, apperrors.Wrap(fmt.Errorf("%w: %s (%s, %s)", err, query, streamID.String(), streamName))
	}
	defer rows.Close()

	var events []*domain.Event

	for rows.Next() {
		var (
			event    domain.Event
			id       string
			payload  json.RawMessage
			metadata sql.NullString
		)
		if err := rows.Scan(
			&id,
			&event.StreamName,
			&event.StreamVersion,
			&event.OccurredAt,
			payload,
			&metadata,
		); err != nil {
			return nil, apperrors.Wrap(err)
		}

		event.Payload, err = getRawEvent(event.Type, payload)
		if err != nil {
			return nil, apperrors.Wrap(err)
		}
		event.Metadata, err = getEventMetadata(json.RawMessage(metadata.String))
		if err != nil {
			return nil, apperrors.Wrap(err)
		}

		event.ID = uuid.MustParse(id)
		event.StreamID = streamID
		event.Type = eventType

		events = append(events, &event)
	}

	if err := rows.Err(); err != nil {
		return nil, apperrors.Wrap(err)
	}

	return events, nil
}

func getRawEvent(eventType string, data json.RawMessage) (domain.RawEvent, error) {
	rawEvent, err := domain.NewRawEvent(eventType)
	if err != nil {
		return nil, apperrors.Wrap(fmt.Errorf("failed to create raw event: %s: %w", eventType, err))
	}
	if err := json.Unmarshal(data, &rawEvent); err != nil {
		return nil, apperrors.Wrap(fmt.Errorf("failed to unmarshal raw event: %s: %w", eventType, err))
	}

	e, ok := rawEvent.(domain.RawEvent)
	if !ok {
		return nil, apperrors.Wrap(fmt.Errorf("aw event does not implement domain.RawEvent: %s: %w", eventType, err))
	}

	return e, nil
}

func getEventMetadata(data json.RawMessage) (*domain.EventMetadata, error) {
	if len(data) == 0 {
		return nil, nil
	}
	var meta domain.EventMetadata
	if err := json.Unmarshal(data, &meta); err != nil {
		return nil, apperrors.Wrap(fmt.Errorf("failed to unmarshal event meta: %w", err))
	}
	return &meta, nil
}
