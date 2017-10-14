package auth

import (
	"encoding/json"

	"github.com/google/uuid"
)

type Identity struct {
	Id    uuid.UUID `json:"id"`
	Email string    `json:"email"`
	Roles []string  `json:"roles"`
}

func NewIdentityFromGoogleData(data json.RawMessage) *Identity {
	return &Identity{}
}

func NewIdentityFromFacebookData(data json.RawMessage) *Identity {
	return &Identity{}
}

func NewUserIdentity(id uuid.UUID, email string) *Identity {
	return &Identity{id, email, nil}
}
