package token

import (
	"fmt"

	"github.com/google/uuid"
)

// WasCreated event
type WasCreated struct {
	ID       uuid.UUID `json:"id"`
	ClientID uuid.UUID `json:"client_id"`
	UserID   uuid.UUID `json:"user_id"`
	Code     string    `json:"code"`
	Scope    string    `json:"scope"`
	Access   string    `json:"access"`
	Refresh  string    `json:"refresh"`
}

// GetType returns event type
func (e WasCreated) GetType() string {
	return fmt.Sprintf("%T", e)
}

// GetID the id
func (e WasCreated) GetID() string {
	return e.ID.String()
}

// GetClientID the client id
func (e WasCreated) GetClientID() string {
	return e.ClientID.String()
}

// GetUserID the user id
func (e WasCreated) GetUserID() string {
	return e.UserID.String()
}

// GetAccess access token
func (e WasCreated) GetAccess() string {
	return e.Access
}

// GetRefresh refresh token
func (e WasCreated) GetRefresh() string {
	return e.Refresh
}

// GetScope get scope of authorization
func (e WasCreated) GetScope() string {
	return e.Scope
}

// GetCode authorization code
func (e WasCreated) GetCode() string {
	return e.Code
}

// WasRemoved event
type WasRemoved struct {
	ID uuid.UUID `json:"id"`
}

// GetType returns event type
func (e WasRemoved) GetType() string {
	return fmt.Sprintf("%T", e)
}
