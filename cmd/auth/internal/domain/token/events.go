package token

import (
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"gopkg.in/oauth2.v4"
	"gopkg.in/oauth2.v4/models"
)

var (
	WasCreatedType = (WasCreated{}).GetType()
	WasRemovedType = (WasRemoved{}).GetType()
)

// WasCreated event
type WasCreated struct {
	ID        uuid.UUID       `json:"id" bson:"id"`
	ClientID  uuid.UUID       `json:"client_id" bson:"client_id"`
	UserID    uuid.UUID       `json:"user_id" bson:"user_id"`
	Data      json.RawMessage `json:"data" bson:"data"`
	UserAgent string          `json:"user_agent" bson:"user_agent"`
}

// GetType returns event type
func (e WasCreated) GetType() string {
	return fmt.Sprintf("%T", e)
}

// GetID the id
func (e *WasCreated) GetID() string {
	return e.ID.String()
}

func (e *WasCreated) GetUserAgent() string {
	return e.UserAgent
}

func (e *WasCreated) GetData() json.RawMessage {
	return e.Data
}

func (e *WasCreated) TokenInfo() (oauth2.TokenInfo, error) {
	var tm models.Token
	if err := json.Unmarshal(e.Data, &tm); err != nil {
		return &tm, err
	}
	return &tm, nil
}

// WasRemoved event
type WasRemoved struct {
	ID uuid.UUID `json:"id" bson:"id"`
}

// GetType returns event type
func (e WasRemoved) GetType() string {
	return fmt.Sprintf("%T", e)
}
