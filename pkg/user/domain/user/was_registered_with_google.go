package user

import (
	"github.com/google/uuid"
)

// WasRegisteredWithGoogle event
type WasRegisteredWithGoogle struct {
	ID        uuid.UUID `json:"id"`
	Email     string    `json:"email"`
	AuthToken string    `json:"authToken"`
}
