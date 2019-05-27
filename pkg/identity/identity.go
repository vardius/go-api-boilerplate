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
	Email string    `json:"email"`
	Roles []string  `json:"roles"`
}

// WithEmail returns a new Identity with given email value
func WithEmail(email string) (*Identity, error) {
	i, err := New()
	if err != nil {
		return nil, err
	}

	i.Email = email

	return i, nil
}

// WithValues returns a new Identity for given values
func WithValues(id uuid.UUID, email string, roles []string) *Identity {
	return &Identity{id, email, roles}
}

// New returns a new Identity
func New() (*Identity, error) {
	id, err := uuid.NewRandom()

	return &Identity{
		ID: id,
	}, err
}
