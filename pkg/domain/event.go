package domain

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/vardius/go-api-boilerplate/pkg/identity"
)

// NullEvent represents empty event
var NullEvent Event

// RawEvent represents raw event that it is aware of its type
type RawEvent interface {
	GetType() string
}

// Event contains id, payload and metadata
type Event struct {
	ID       uuid.UUID          `json:"id"`
	Metadata EventMetaData      `json:"metadata"`
	Payload  json.RawMessage    `json:"payload"`
	Identity *identity.Identity `json:"identity,omitempty"`
}

// EventMetaData for Event
type EventMetaData struct {
	Type          string    `json:"type"`
	StreamID      uuid.UUID `json:"stream_id"`
	StreamName    string    `json:"stream_name"`
	StreamVersion int       `json:"stream_version"`
	OccurredAt    time.Time `json:"occurred_at"`
}

// NewEvent create new event
func NewEvent(streamID uuid.UUID, streamName string, streamVersion int, rawEvent RawEvent, identity *identity.Identity) (Event, error) {
	meta := EventMetaData{
		Type:          rawEvent.GetType(),
		StreamID:      streamID,
		StreamName:    streamName,
		StreamVersion: streamVersion,
		OccurredAt:    time.Now(),
	}

	payload, err := json.Marshal(rawEvent)
	if err != nil {
		return NullEvent, fmt.Errorf("could not parse event to json: %w", err)
	}

	id, err := uuid.NewRandom()
	if err != nil {
		return NullEvent, fmt.Errorf("could not generate event id: %w", err)
	}

	return Event{
		ID:       id,
		Metadata: meta,
		Payload:  payload,
		Identity: identity,
	}, nil
}

// MakeEvent makes a event object from metadata and payload
func MakeEvent(meta EventMetaData, payload json.RawMessage, identity *identity.Identity) (Event, error) {
	id, err := uuid.NewRandom()
	if err != nil {
		return NullEvent, fmt.Errorf("could not generate event id: %w", err)
	}

	return Event{
		ID:       id,
		Metadata: meta,
		Payload:  payload,
		Identity: identity,
	}, nil
}
