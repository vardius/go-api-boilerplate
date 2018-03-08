package user

import (
	"github.com/google/uuid"
)

// EmailAddressWasChanged event
type EmailAddressWasChanged struct {
	ID    uuid.UUID `json:"id"`
	Email string    `json:"email"`
}
