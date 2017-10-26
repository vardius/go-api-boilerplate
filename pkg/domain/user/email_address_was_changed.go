package user

import (
	"app/pkg/domain"
	"context"

	"github.com/google/uuid"
)

type EmailAddressWasChanged struct {
	ID    uuid.UUID `json:"id"`
	Email string    `json:"email"`
}

func onEmailAddressWasChanged(ctx context.Context, event domain.Event) {
	// todo: register user
}
