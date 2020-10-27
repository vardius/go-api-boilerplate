package domain

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// NullEvent represents empty event
var NullEvent Event

// RawEvent represents raw event that it is aware of its type
type RawEvent interface {
	GetType() string
}

// Event contains id, payload and metadata
type Event struct {
	ID            uuid.UUID       `json:"id" bson:"event_id"`
	Type          string          `json:"type" bson:"type"`
	StreamID      uuid.UUID       `json:"stream_id" bson:"stream_id"`
	StreamName    string          `json:"stream_name" bson:"stream_name"`
	StreamVersion int             `json:"stream_version" bson:"stream_version"`
	OccurredAt    time.Time       `json:"occurred_at" bson:"occurred_at"`
	Payload       json.RawMessage `json:"payload" bson:"payload"`
	Metadata      json.RawMessage `json:"metadata,omitempty" bson:"metadata,omitempty"`
}

// NewEventFromRawEvent create new event
func NewEventFromRawEvent(streamID uuid.UUID, streamName string, streamVersion int, rawEvent RawEvent) (Event, error) {
	payload, err := json.Marshal(rawEvent)
	if err != nil {
		return NullEvent, fmt.Errorf("could not parse event to json: %w", err)
	}

	id, err := uuid.NewRandom()
	if err != nil {
		return NullEvent, fmt.Errorf("could not generate event id: %w", err)
	}

	return Event{
		ID:            id,
		Type:          rawEvent.GetType(),
		StreamID:      streamID,
		StreamName:    streamName,
		StreamVersion: streamVersion,
		OccurredAt:    time.Now(),
		Payload:       payload,
	}, nil
}

// NewEventFromPayload makes a event object from metadata and payload
func NewEventFromPayload(streamID uuid.UUID, streamName string, streamVersion int, eventID uuid.UUID, eventType string, occurredAt time.Time, payload json.RawMessage) (Event, error) {
	return Event{
		ID:            eventID,
		Type:          eventType,
		StreamID:      streamID,
		StreamName:    streamName,
		StreamVersion: streamVersion,
		OccurredAt:    occurredAt,
		Payload:       payload,
	}, nil
}

func (e *Event) WithMetadata(rawMeta interface{}) error {
	meta, err := json.Marshal(rawMeta)
	if err != nil {
		return fmt.Errorf("could not parse event metadata to json: %w", err)
	}

	e.Metadata = meta

	return nil
}
