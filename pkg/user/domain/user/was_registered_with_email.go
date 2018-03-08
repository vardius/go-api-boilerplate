package user

import (
	"github.com/google/uuid"
)

// WasRegisteredWithEmail event
type WasRegisteredWithEmail struct {
	ID        uuid.UUID `json:"id"`
	Email     string    `json:"email"`
	AuthToken string    `json:"authToken"`
}
