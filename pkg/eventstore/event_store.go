package eventstore

import (
	"context"

	"github.com/google/uuid"

	"github.com/vardius/go-api-boilerplate/pkg/domain"
)

// EventStore methods allow to save, load events and event streams
type EventStore interface {
	Store(ctx context.Context, events []*domain.Event) error
	Get(ctx context.Context, id uuid.UUID) (*domain.Event, error)
	FindAll(ctx context.Context) ([]*domain.Event, error)
	GetStream(ctx context.Context, streamID uuid.UUID, streamName string) ([]*domain.Event, error)
	GetStreamEventsByType(ctx context.Context, streamID uuid.UUID, streamName, eventType string) ([]*domain.Event, error)
}
