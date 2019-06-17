/*
Package identity provides type that allows to authorize request
*/
package identity

import (
	"github.com/google/uuid"
)

// Identity data to be encode in auth token
type Identity struct {
	ID    uuid.UUID `json:"id"`
	Token string    `json:"token"`
	Email string    `json:"email"`
	Roles []string  `json:"roles"`
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

// New returns a new Identity
func New(id, token, email string, roles []string) Identity {
	return Identity{uuid.MustParse(id), token, email, roles}
}
