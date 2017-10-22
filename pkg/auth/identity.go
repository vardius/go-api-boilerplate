package auth

import (
	"encoding/json"
	"errors"

	"github.com/google/uuid"
)

type Identity struct {
	Id    uuid.UUID `json:"id"`
	Email string    `json:"email"`
	Roles []string  `json:"roles"`
}

// FromGoogleData sets *i to a copy of data.
func (i *Identity) FromGoogleData(data json.RawMessage) error {
	if i == nil {
		return errors.New("auth.Identity: FromGoogleData on nil pointer")
	}
	//todo set props from google data
	return nil
}

// FromFacebookData sets *i to a copy of data.
func (i *Identity) FromFacebookData(data json.RawMessage) error {
	if i == nil {
		return errors.New("auth.Identity: FromFacebookData on nil pointer")
	}
	//todo set props from facebook data
	return nil
}

// NewUserIdentity returns a new Identity with empty roles slice.
func NewUserIdentity(id uuid.UUID, email string) *Identity {
	return &Identity{id, email, nil}
}
