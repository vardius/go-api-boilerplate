package token

import (
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"gopkg.in/oauth2.v4"
	"gopkg.in/oauth2.v4/models"
)

// WasCreated event
type WasCreated struct {
	ID        uuid.UUID       `json:"id"`
	ClientID  uuid.UUID       `json:"client_id"`
	UserID    uuid.UUID       `json:"user_id"`
	Data      json.RawMessage `json:"data"`
	UserAgent string          `json:"user_agent"`
}

// GetType returns event type
func (e WasCreated) GetType() string {
	return fmt.Sprintf("%T", e)
}

// GetID the id
func (e WasCreated) GetID() string {
	return e.ID.String()
}

func (e WasCreated) GetUserAgent() string {
	return e.UserAgent
}

func (e WasCreated) GetData() json.RawMessage {
	return e.Data
}

func (e WasCreated) TokenInfo() (oauth2.TokenInfo, error) {
	var tm models.Token
	if err := json.Unmarshal(e.Data, &tm); err != nil {
		return &tm, err
	}
	return &tm, nil
}

// WasRemoved event
type WasRemoved struct {
	ID uuid.UUID `json:"id"`
}

// GetType returns event type
func (e WasRemoved) GetType() string {
	return fmt.Sprintf("%T", e)
}
