package user

import (
	"context"
	"log"

	"github.com/google/uuid"
	"github.com/vardius/go-api-boilerplate/pkg/domain"
)

// EmailAddressWasChanged event
type EmailAddressWasChanged struct {
	ID    uuid.UUID `json:"id"`
	Email string    `json:"email"`
}

// WhenEmailAddressWasChanged handles event
func WhenEmailAddressWasChanged(ctx context.Context, event domain.Event) {
	// todo: register user
	log.Printf("handle %v", event)
}
