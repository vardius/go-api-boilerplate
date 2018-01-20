package user

import (
	"context"
	"log"

	"github.com/google/uuid"
	"github.com/vardius/go-api-boilerplate/pkg/domain"
)

// WasRegisteredWithFacebook event
type WasRegisteredWithFacebook struct {
	ID        uuid.UUID `json:"id"`
	Email     string    `json:"email"`
	AuthToken string    `json:"authToken"`
}

// WhenWasRegisteredWithFacebook handles event
func WhenWasRegisteredWithFacebook(ctx context.Context, event domain.Event) {
	// todo: register user
	log.Printf("handle %v", event)
}
