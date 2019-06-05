package client

import (
	"fmt"

	"github.com/google/uuid"
	oauth2 "gopkg.in/oauth2.v3"
)

// WasCreated event
type WasCreated struct {
	ID     uuid.UUID `json:"id"`
	UserID uuid.UUID `json:"userId"`

	Info oauth2.ClientInfo `json:"data"`
}

// GetType returns event type
func (e *WasCreated) GetType() string {
	return fmt.Sprintf("%T", e)
}

// WasRemoved event
type WasRemoved struct {
	ID uuid.UUID `json:"id"`
}

// GetType returns event type
func (e *WasRemoved) GetType() string {
	return fmt.Sprintf("%T", e)
}
