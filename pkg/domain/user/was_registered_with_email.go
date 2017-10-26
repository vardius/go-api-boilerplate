package user

import (
	"app/pkg/domain"
	"context"

	"github.com/google/uuid"
)

type WasRegisteredWithEmail struct {
	ID        uuid.UUID `json:"id"`
	Email     string    `json:"email"`
	AuthToken string    `json:"authToken"`
}

func onWasRegisteredWithEmail(ctx context.Context, event domain.Event) {
	// todo: register user and send email with auth token
}
