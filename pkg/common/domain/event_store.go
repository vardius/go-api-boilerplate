package domain

import "github.com/google/uuid"

// EventStore methods allow to save, load events and event streams
type EventStore interface {
	Store([]*Event) error
	Get(uuid.UUID) (*Event, error)
	FindAll() []*Event
	GetStream(uuid.UUID, string) []*Event
}
