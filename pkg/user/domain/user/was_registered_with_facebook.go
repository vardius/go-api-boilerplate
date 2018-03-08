package user

import (
	"github.com/google/uuid"
)

// WasRegisteredWithFacebook event
type WasRegisteredWithFacebook struct {
	ID        uuid.UUID `json:"id"`
	Email     string    `json:"email"`
	AuthToken string    `json:"authToken"`
}
