/*
Package mysql holds view model repositories
*/
package mysql

import (
	"encoding/json"

	"github.com/vardius/go-api-boilerplate/pkg/mysql"
	oauth2 "gopkg.in/oauth2.v3"
)

// Token model
type Token struct {
	ID       string           `json:"id"`
	ClientID string           `json:"clientId"`
	UserID   string           `json:"userId"`
	Scope    string           `json:"scope"`
	Access   string           `json:"access"`
	Refresh  string           `json:"refresh"`
	Code     mysql.NullString `json:"code"`
	Data     json.RawMessage  `json:"data"`
}

// GetInfo token oauth2 info
func (t *Token) GetInfo() (i oauth2.TokenInfo) {
	json.Unmarshal(t.Data, &i)

	return i
}

// GetID the id
func (t *Token) GetID() string {
	return t.ID
}

// GetClientID the client id
func (t *Token) GetClientID() string {
	return t.ClientID
}

// GetUserID the user id
func (t *Token) GetUserID() string {
	return t.UserID
}

// GetScope get scope of authorization
func (t *Token) GetScope() string {
	return t.Scope
}

// GetAccess access Token
func (t *Token) GetAccess() string {
	return t.Access
}

// GetRefresh refresh Token
func (t *Token) GetRefresh() string {
	return t.Refresh
}

// GetCode authorization code
func (t *Token) GetCode() string {
	return t.Code.String
}

// GetData token data
func (t *Token) GetData() json.RawMessage {
	return t.Data
}
