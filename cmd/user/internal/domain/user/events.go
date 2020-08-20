package user

import (
	"fmt"

	"github.com/google/uuid"
)

// AccessTokenWasRequested event
type AccessTokenWasRequested struct {
	ID    uuid.UUID    `json:"id"`
	Email EmailAddress `json:"email"`
}

// GetType returns event type
func (e AccessTokenWasRequested) GetType() string {
	return fmt.Sprintf("%T", e)
}

// EmailAddressWasChanged event
type EmailAddressWasChanged struct {
	ID    uuid.UUID    `json:"id"`
	Email EmailAddress `json:"email"`
}

// GetType returns event type
func (e EmailAddressWasChanged) GetType() string {
	return fmt.Sprintf("%T", e)
}

// WasRegisteredWithEmail event
type WasRegisteredWithEmail struct {
	ID    uuid.UUID    `json:"id"`
	Email EmailAddress `json:"email"`
}

// GetType returns event type
func (e WasRegisteredWithEmail) GetType() string {
	return fmt.Sprintf("%T", e)
}

// WasRegisteredWithFacebook event
type WasRegisteredWithFacebook struct {
	ID         uuid.UUID    `json:"id"`
	Email      EmailAddress `json:"email"`
	FacebookID string       `json:"facebook_id"`
}

// GetType returns event type
func (e WasRegisteredWithFacebook) GetType() string {
	return fmt.Sprintf("%T", e)
}

// ConnectedWithFacebook event
type ConnectedWithFacebook struct {
	ID         uuid.UUID `json:"id"`
	FacebookID string    `json:"facebook_id"`
}

// GetType returns event type
func (e ConnectedWithFacebook) GetType() string {
	return fmt.Sprintf("%T", e)
}

// WasRegisteredWithGoogle event
type WasRegisteredWithGoogle struct {
	ID       uuid.UUID    `json:"id"`
	Email    EmailAddress `json:"email"`
	GoogleID string       `json:"google_id"`
}

// GetType returns event type
func (e WasRegisteredWithGoogle) GetType() string {
	return fmt.Sprintf("%T", e)
}

// ConnectedWithGoogle event
type ConnectedWithGoogle struct {
	ID       uuid.UUID `json:"id"`
	GoogleID string    `json:"google_id"`
}

// GetType returns event type
func (e ConnectedWithGoogle) GetType() string {
	return fmt.Sprintf("%T", e)
}
