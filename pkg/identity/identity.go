/*
Package identity provides type that allows to authorize request
*/
package identity

import (
	"github.com/google/uuid"
)

// NullIdentity represents empty Identity
var NullIdentity = Identity{}

// Role type
type Role uint8

// Roles
const (
	// @TODO: MANAGE YOUR ROLES HERE
	RoleUser Role = 1 << iota
	RoleAdmin
)

func (r Role) String() string {
	return [...]string{"USER", "ADMIN"}[r]
}

// Identity data to be encode in auth token
type Identity struct {
	ID    uuid.UUID `json:"id"`
	Token string    `json:"token"`
	Email string    `json:"email"`
	Roles Role      `json:"roles"`
}

// WithEmail returns copy of an identity with given email value
func (i Identity) WithEmail(email string) Identity {
	i.Email = email

	return i
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
func New(id uuid.UUID, token, email string) Identity {
	return Identity{id, token, email, RoleUser}
}
