/*
Package identity provides type that allows to authorize request
*/
package identity

import (
	"github.com/google/uuid"
)

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
	ClientSecret uuid.UUID `json:"client_secret"`
	ClientDomain string    `json:"client_domain"`
	Roles        Role      `json:"roles"`
}

// WithToken returns copy of an identity with given oauth2 token
func (i *Identity) WithToken(token string) {
	i.Token = token
}

// WithRole adds role to identity roles
func (i *Identity) WithRole(role Role) {
	i.Roles |= role
}

// RemoveRole removes role from identity
func (i *Identity) RemoveRole(role Role) {
	i.Roles = i.Roles &^ role
}

// HasRole returns true if identity has give role
func (i *Identity) HasRole(role Role) bool { return i.Roles&role != 0 }

// New returns a new Identity
func New(userID, clientID, clientSecret uuid.UUID, userEmail, token string) *Identity {
	return &Identity{
		Token:        token,
		UserID:       userID,
		UserEmail:    userEmail,
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Roles:        RoleUser,
	}
}
