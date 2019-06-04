package client

import (
	"github.com/google/uuid"
	oauth2 "gopkg.in/oauth2.v3"
)

// WasCreated event
type WasCreated struct {
	ID     uuid.UUID `json:"id"`
	UserID uuid.UUID `json:"userId"`

	Info oauth2.ClientInfo `json:"data"`
}

// WasRemoved event
type WasRemoved struct {
	ID uuid.UUID `json:"id"`
}
