package user

import (
	"github.com/google/uuid"
)

// EmailAddressWasChanged event
type EmailAddressWasChanged struct {
	ID    uuid.UUID `json:"id"`
	Email string    `json:"email"`
}

// WasRegisteredWithEmail event
type WasRegisteredWithEmail struct {
	ID    uuid.UUID `json:"id"`
	Email string    `json:"email"`
}

// WasRegisteredWithFacebook event
type WasRegisteredWithFacebook struct {
	ID         uuid.UUID `json:"id"`
	Email      string    `json:"email"`
	FacebookID string    `json:"facebookId"`
}

// ConnectedWithFacebook event
type ConnectedWithFacebook struct {
	ID         uuid.UUID `json:"id"`
	FacebookID string    `json:"facebookId"`
}

// WasRegisteredWithGoogle event
type WasRegisteredWithGoogle struct {
	ID       uuid.UUID `json:"id"`
	Email    string    `json:"email"`
	GoogleID string    `json:"googleId"`
}

// ConnectedWithGoogle event
type ConnectedWithGoogle struct {
	ID       uuid.UUID `json:"id"`
	GoogleID string    `json:"googleId"`
}
