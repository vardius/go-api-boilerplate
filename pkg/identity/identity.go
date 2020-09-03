/*
Package identity provides type that allows to authorize request
*/
package identity

import (
	"github.com/google/uuid"
)

// NullIdentity represents empty Identity
var NullIdentity Identity

// Role type
type Role uint8

// Roles
const (
	// @TODO: MANAGE YOUR ROLES HERE
	RoleUser Role = 1 << iota
	RoleAdmin
	RoleSuperAdmin
)

func (r Role) String() string {
	return [...]string{"USER", "ADMIN", "SUPER_ADMIN"}[r>>1]
}

// Identity data to be encode in auth token
type Identity struct {
	Token        string    `json:"token"`
	UserID       uuid.UUID `json:"user_id"`
	UserEmail    string    `json:"user_email"`
	ClientID     uuid.UUID `json:"client_id"`
	ClientSecret string    `json:"client_secret"`
	Roles        Role      `json:"roles"`
}

// WithToken returns copy of an identity with given oauth2 token
func (i Identity) WithToken(token string) Identity {
	i.Token = token

	return i
}

// WithRole adds role to identity roles
func (i Identity) WithRole(role Role) Identity {
	i.Roles |= role

	return i
}

// RemoveRole removes role from identity
func (i Identity) RemoveRole(role Role) Identity {
	i.Roles = i.Roles &^ role

	return i
}

// HasRole returns true if identity has give role
func (i Identity) HasRole(role Role) bool { return i.Roles&role != 0 }

// New returns a new Identity
func New(userID, clientID uuid.UUID, userEmail, clientSecret, token string) Identity {
	return Identity{
		Token:        token,
		UserID:       userID,
		UserEmail:    userEmail,
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Roles:        RoleUser,
	}
}
