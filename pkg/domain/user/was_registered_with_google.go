package user

import (
	"app/pkg/domain"
	"context"

	"github.com/google/uuid"
)

type WasRegisteredWithGoogle struct {
	ID        uuid.UUID `json:"id"`
	Email     string    `json:"email"`
	AuthToken string    `json:"authToken"`
}

func onWasRegisteredWithGoogle(ctx context.Context, event domain.Event) {
	// todo: register user
}
