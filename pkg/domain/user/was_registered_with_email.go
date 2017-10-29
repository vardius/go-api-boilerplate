package user

import (
	"github.com/vardius/go-api-boilerplate/pkg/domain"
	"context"
	"log"

	"github.com/google/uuid"
)

// WasRegisteredWithEmail event
type WasRegisteredWithEmail struct {
	ID        uuid.UUID `json:"id"`
	Email     string    `json:"email"`
	AuthToken string    `json:"authToken"`
}

func onWasRegisteredWithEmail(ctx context.Context, event domain.Event) {
	// todo: register user and send email with auth token
	log.Printf("handle %v", event)
}
