package domain

import (
	"encoding/json"
	"reflect"
	"time"

	"github.com/google/uuid"
)

type Event struct {
	Id       uuid.UUID       `json:"id"`
	Metadata EventMetaData   `json:"metadata"`
	Payload  json.RawMessage `json:"payload"`
}

type EventMetaData struct {
	Type          string    `json:"type"`
	StreamId      uuid.UUID `json:"stream_id"`
	StreamName    string    `json:"stream_name"`
	StreamVersion int       `json:"stream_version"`
	OccuredAt     time.Time `json:"occured_at"`
}

//Create new event
func NewEvent(streamId uuid.UUID, streamName string, streamVersion int, data interface{}) (*Event, error) {
	id, err := uuid.NewRandom()
	if err != nil {
		return nil, nil
	}

	meta := EventMetaData{
		Type:          reflect.TypeOf(data).String(),
		StreamId:      streamId,
		StreamName:    streamName,
		StreamVersion: streamVersion,
		OccuredAt:     time.Now(),
	}

	payload, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	return &Event{id, meta, payload}, nil
}

//Makes a event object from metadata and payload
func MakeEvent(meta EventMetaData, payload json.RawMessage) (*Event, error) {
	id, err := uuid.NewRandom()
	if err != nil {
		return nil, err
	}

	return &Event{id, meta, payload}, nil
}
