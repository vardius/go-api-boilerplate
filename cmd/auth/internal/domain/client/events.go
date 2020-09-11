package client

import (
	"fmt"

	"github.com/google/uuid"
)

// WasCreated event
type WasCreated struct {
	ID          uuid.UUID `json:"id"`
	UserID      uuid.UUID `json:"user_id"`
	Secret      uuid.UUID `json:"secret"`
	Domain      string    `json:"domain"`
	RedirectURL string    `json:"redirect_url"`
	Scopes      []string  `json:"scopes"`
}

// GetType returns event type
func (e WasCreated) GetType() string {
	return fmt.Sprintf("%T", e)
}

// GetID client id
func (e WasCreated) GetID() string {
	return e.ID.String()
}

// GetSecret client domain
func (e WasCreated) GetSecret() string {
	return e.Secret.String()
}

// GetDomain client domain
func (e WasCreated) GetDomain() string {
	return e.Domain
}

// GetUserID user id
func (e WasCreated) GetUserID() string {
	return e.UserID.String()
}

// GetRedirectURL user id
func (e WasCreated) GetRedirectURL() string {
	return e.RedirectURL
}

// GetScopes user id
func (e WasCreated) GetScopes() []string {
	return e.Scopes
}

// WasRemoved event
type WasRemoved struct {
	ID uuid.UUID `json:"id"`
}

// GetType returns event type
func (e WasRemoved) GetType() string {
	return fmt.Sprintf("%T", e)
}
