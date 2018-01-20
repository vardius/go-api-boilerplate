package user

import (
	"context"
	"log"

	"github.com/google/uuid"
	"github.com/vardius/go-api-boilerplate/pkg/domain"
)

// WasRegisteredWithGoogle event
type WasRegisteredWithGoogle struct {
	ID        uuid.UUID `json:"id"`
	Email     string    `json:"email"`
	AuthToken string    `json:"authToken"`
}

func WhenWasRegisteredWithGoogle(ctx context.Context, event domain.Event) {
	// todo: register user
	log.Printf("handle %v", event)
}
