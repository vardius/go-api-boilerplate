package user

import (
	"fmt"

	"github.com/google/uuid"
)

// AccessTokenWasRequested event
type AccessTokenWasRequested struct {
	ID           uuid.UUID    `json:"id"`
	Email        EmailAddress `json:"email"`
	RedirectPath string       `json:"redirect_path,omitempty"`
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
	ID           uuid.UUID    `json:"id"`
	Email        EmailAddress `json:"email"`
	RedirectPath string       `json:"redirect_path,omitempty"`
}

// GetType returns event type
func (e WasRegisteredWithEmail) GetType() string {
	return fmt.Sprintf("%T", e)
}

// GetID the id
func (e WasRegisteredWithEmail) GetID() string {
	return e.ID.String()
}

// GetEmail the email
func (e WasRegisteredWithEmail) GetEmail() string {
	return e.Email.String()
}

// GetFacebookID facebook id
func (e WasRegisteredWithEmail) GetFacebookID() string {
	return ""
}

// GetGoogleID google id
func (e WasRegisteredWithEmail) GetGoogleID() string {
	return ""
}

// WasRegisteredWithFacebook event
type WasRegisteredWithFacebook struct {
	ID          uuid.UUID    `json:"id"`
	Email       EmailAddress `json:"email"`
	FacebookID  string       `json:"facebook_id"`
	AccessToken string       `json:"access_token"`
}

// GetType returns event type
func (e WasRegisteredWithFacebook) GetType() string {
	return fmt.Sprintf("%T", e)
}

// GetID the id
func (e WasRegisteredWithFacebook) GetID() string {
	return e.ID.String()
}

// GetEmail the email
func (e WasRegisteredWithFacebook) GetEmail() string {
	return e.Email.String()
}

// GetFacebookID facebook id
func (e WasRegisteredWithFacebook) GetFacebookID() string {
	return e.FacebookID
}

// GetGoogleID google id
func (e WasRegisteredWithFacebook) GetGoogleID() string {
	return ""
}

// ConnectedWithFacebook event
type ConnectedWithFacebook struct {
	ID          uuid.UUID `json:"id"`
	FacebookID  string    `json:"facebook_id"`
	AccessToken string    `json:"access_token"`
}

// GetType returns event type
func (e ConnectedWithFacebook) GetType() string {
	return fmt.Sprintf("%T", e)
}

// WasRegisteredWithGoogle event
type WasRegisteredWithGoogle struct {
	ID          uuid.UUID    `json:"id"`
	Email       EmailAddress `json:"email"`
	GoogleID    string       `json:"google_id"`
	AccessToken string       `json:"access_token"`
}

// GetType returns event type
func (e WasRegisteredWithGoogle) GetType() string {
	return fmt.Sprintf("%T", e)
}

// GetID the id
func (e WasRegisteredWithGoogle) GetID() string {
	return e.ID.String()
}

// GetEmail the email
func (e WasRegisteredWithGoogle) GetEmail() string {
	return e.Email.String()
}

// GetFacebookID facebook id
func (e WasRegisteredWithGoogle) GetFacebookID() string {
	return ""
}

// GetGoogleID google id
func (e WasRegisteredWithGoogle) GetGoogleID() string {
	return e.GoogleID
}

// ConnectedWithGoogle event
type ConnectedWithGoogle struct {
	ID          uuid.UUID `json:"id"`
	GoogleID    string    `json:"google_id"`
	AccessToken string    `json:"access_token"`
}

// GetType returns event type
func (e ConnectedWithGoogle) GetType() string {
	return fmt.Sprintf("%T", e)
}
