package user

import (
	"app/pkg/domain"
	"context"

	"github.com/google/uuid"
)

type WasRegisteredWithFacebook struct {
	ID        uuid.UUID `json:"id"`
	Email     string    `json:"email"`
	AuthToken string    `json:"authToken"`
}

func onWasRegisteredWithFacebook(ctx context.Context, event domain.Event) {
	// todo: register user
}
