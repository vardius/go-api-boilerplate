/*
Package identity provides type that allows to authorize request
*/
package identity

import (
	"github.com/google/uuid"
)

// Identity data to be encode in auth token
type Identity struct {
	Token        string     `json:"token"`
	Permission   Permission `json:"permission"`
	UserID       uuid.UUID  `json:"user_id"`
	ClientID     uuid.UUID  `json:"client_id,omitempty"`
	ClientDomain string     `json:"client_domain,omitempty"`
}

// Flag type
type Permission uint8

// Add permission
func (p Permission) Add(flag Permission) Permission { return p | flag }

// Remove permission
func (p Permission) Remove(flag Permission) Permission { return p &^ flag }

// Has permission
func (p Permission) Has(flag Permission) bool { return p&flag != 0 }

// Execution context flags
const (
	PermissionUserRead Permission = 1 << iota
	PermissionUserWrite
	PermissionClientWrite
	PermissionClientRead
	PermissionTokenRead
)
