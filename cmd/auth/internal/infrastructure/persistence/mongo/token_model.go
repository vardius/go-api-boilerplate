package mongo

import (
	"encoding/json"
	"time"

	"gopkg.in/oauth2.v4"
	"gopkg.in/oauth2.v4/models"
)

// Token model
type Token struct {
	ID        string          `json:"id" bson:"token_id"`
	ClientID  string          `json:"-" bson:"client_id"`
	UserID    string          `json:"-" bson:"user_id"`
	Access    string          `json:"access" bson:"access"`
	Refresh   string          `json:"refresh,omitempty" bson:"refresh,omitempty"`
	Code      string          `json:"-" bson:"code"`
	UserAgent string          `json:"user_agent,omitempty" bson:"user_agent,omitempty"`
	ExpiredAt *time.Time      `json:"-" bson:"expiredAt"`
	Data      json.RawMessage `json:"-" bson:"data"`
}

func (t *Token) GetID() string {
	return t.ID
}

func (t *Token) GetUserAgent() string {
	return t.UserAgent
}

func (t *Token) GetData() json.RawMessage {
	return t.Data
}

func (t *Token) TokenInfo() (oauth2.TokenInfo, error) {
	var tm models.Token
	if err := json.Unmarshal(t.Data, &tm); err != nil {
		return &tm, err
	}
	return &tm, nil
}
