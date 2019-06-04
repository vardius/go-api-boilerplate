package token

import (
	"github.com/google/uuid"
	oauth2 "gopkg.in/oauth2.v3"
)

// WasCreated event
type WasCreated struct {
	ID uuid.UUID `json:"id"`

	ClientID uuid.UUID `json:"clientId"`
	UserID   uuid.UUID `json:"userId"`
	Code     string    `json:"code"`
	Access   string    `json:"access"`
	Refresh  string    `json:"refresh"`

	Info oauth2.TokenInfo `json:"data"`
}

// WasRemoved event
type WasRemoved struct {
	ID uuid.UUID `json:"id"`
}
