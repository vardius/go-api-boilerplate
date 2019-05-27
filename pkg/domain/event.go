package domain

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Event contains id, payload and metadata
type Event struct {
	ID       uuid.UUID       `json:"id"`
	Metadata EventMetaData   `json:"metadata"`
	Payload  json.RawMessage `json:"payload"`
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
func NewEvent(streamID uuid.UUID, streamName string, streamVersion int, data interface{}) (*Event, error) {
	meta := EventMetaData{
		Type:          fmt.Sprintf("%T", data),
		StreamID:      streamID,
		StreamName:    streamName,
		StreamVersion: streamVersion,
		OccurredAt:    time.Now(),
	}

	payload, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	id, err := uuid.NewRandom()

	return &Event{id, meta, payload}, err
}

// MakeEvent makes a event object from metadata and payload
func MakeEvent(meta EventMetaData, payload json.RawMessage) (*Event, error) {
	id, err := uuid.NewRandom()

	return &Event{id, meta, payload}, err
}
