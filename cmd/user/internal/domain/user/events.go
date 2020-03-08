package user

import (
	"fmt"

	"github.com/google/uuid"
)

// AccessTokenWasRequested event
type AccessTokenWasRequested struct {
	ID    uuid.UUID `json:"id"`
	Email string    `json:"email"`
}

// GetType returns event type
func (e AccessTokenWasRequested) GetType() string {
	return fmt.Sprintf("%T", e)
}

// EmailAddressWasChanged event
type EmailAddressWasChanged struct {
	ID    uuid.UUID `json:"id"`
	Email string    `json:"email"`
}

// GetType returns event type
func (e EmailAddressWasChanged) GetType() string {
	return fmt.Sprintf("%T", e)
}

// WasRegisteredWithEmail event
type WasRegisteredWithEmail struct {
	ID           uuid.UUID `json:"id"`
	Provider     string    `json:"provider"`
	Name         string    `json:"name"`
	Email        string    `json:"email"`
	NickName     string    `json:"nickName"`
	Location     string    `json:"location"`
	AvatarURL    string    `json:"avatarURL"`
	Description  string    `json:"description"`
	UserID       string    `json:"userId"`
	RefreshToken string    `json:"refreshToken"`
}

// GetType returns event type
func (e WasRegisteredWithEmail) GetType() string {
	return fmt.Sprintf("%T", e)
}

// WasAuthenticatedWithProvider event
type WasAuthenticatedWithProvider struct {
	ID           uuid.UUID `json:"id"`
	Provider     string    `json:"provider"`
	Name         string    `json:"name"`
	Email        string    `json:"email"`
	NickName     string    `json:"nickName"`
	Location     string    `json:"location"`
	AvatarURL    string    `json:"avatarURL"`
	Description  string    `json:"description"`
	UserID       string    `json:"userId"`
	RefreshToken string    `json:"refreshToken"`
}

// GetType returns event type
func (e WasAuthenticatedWithProvider) GetType() string {
	return fmt.Sprintf("%T", e)
}
