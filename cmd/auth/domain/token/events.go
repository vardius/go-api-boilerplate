package token

import (
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
)

// WasCreated event
type WasCreated struct {
	ID uuid.UUID `json:"id"`

	ClientID uuid.UUID `json:"clientId"`
	UserID   uuid.UUID `json:"userId"`
	Code     string    `json:"code"`
	Scope    string    `json:"scope"`
	Access   string    `json:"access"`
	Refresh  string    `json:"refresh"`

	Data json.RawMessage `json:"data"`
}

// GetType returns event type
func (e WasCreated) GetType() string {
	return fmt.Sprintf("%T", e)
}

// WasRemoved event
type WasRemoved struct {
	ID uuid.UUID `json:"id"`
}

// GetType returns event type
func (e WasRemoved) GetType() string {
	return fmt.Sprintf("%T", e)
}
