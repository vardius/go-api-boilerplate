/*
Package mysql holds view model repositories
*/
package mysql

import (
	"encoding/json"
	"time"

	"gopkg.in/oauth2.v4"
	"gopkg.in/oauth2.v4/models"

	"github.com/vardius/go-api-boilerplate/pkg/mysql"
)

// Token model
type Token struct {
	ID        string           `json:"id"`
	ClientID  string           `json:"-"`
	UserID    string           `json:"-"`
	Access    string           `json:"access"`
	Refresh   mysql.NullString `json:"refresh,omitempty"`
	Code      mysql.NullString `json:"-"`
	UserAgent mysql.NullString `json:"user_agent,omitempty"`
	ExpiredAt time.Time        `json:"-"`
	Data      json.RawMessage  `json:"-"`
}

func (t *Token) GetID() string {
	return t.ID
}

func (t *Token) GetUserAgent() string {
	return t.UserAgent.String
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
