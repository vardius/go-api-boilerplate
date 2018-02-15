package user

import (
	"context"
	"log"

	"github.com/google/uuid"
	"github.com/vardius/go-api-boilerplate/pkg/domain"
)

// WasRegisteredWithEmail event
type WasRegisteredWithEmail struct {
	ID        uuid.UUID `json:"id"`
	Email     string    `json:"email"`
	AuthToken string    `json:"authToken"`
}

// WhenWasRegisteredWithEmail handles event
func WhenWasRegisteredWithEmail(ctx context.Context, event domain.Event) {
	// todo: register user and send email with auth token
	log.Printf("[userserver EventHandler] %s", event.Payload)
}
