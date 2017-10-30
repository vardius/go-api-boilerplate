package user

import (
	"context"
	"github.com/vardius/go-api-boilerplate/pkg/domain"
	"log"

	"github.com/google/uuid"
)

// EmailAddressWasChanged event
type EmailAddressWasChanged struct {
	ID    uuid.UUID `json:"id"`
	Email string    `json:"email"`
}

func onEmailAddressWasChanged(ctx context.Context, event domain.Event) {
	// todo: register user
	log.Printf("handle %v", event)
}
