package domain

import "github.com/google/uuid"

type EventStore interface {
	Store([]*Event) error
	Get(uuid.UUID) (*Event, error)
	FindAll() []*Event
	GetStream(uuid.UUID, string) []*Event
}
