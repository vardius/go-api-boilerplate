package user

import (
	"app/pkg/domain"
	"context"

	"github.com/google/uuid"
)

// WasRegisteredWithFacebook event
type WasRegisteredWithFacebook struct {
	ID        uuid.UUID `json:"id"`
	Email     string    `json:"email"`
	AuthToken string    `json:"authToken"`
}

func onWasRegisteredWithFacebook(ctx context.Context, event domain.Event) {
	// todo: register user
}
