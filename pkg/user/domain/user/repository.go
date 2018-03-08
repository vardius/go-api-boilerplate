package user

import (
	"context"

	"github.com/google/uuid"
)

// EventSourcedRepository allows to get/save events from/to event store
type EventSourcedRepository interface {
	Save(ctx context.Context, u *User) error
	Get(id uuid.UUID) *User
}
