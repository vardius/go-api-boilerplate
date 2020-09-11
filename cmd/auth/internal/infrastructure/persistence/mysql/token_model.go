/*
Package mysql holds view model repositories
*/
package mysql

import (
	"github.com/vardius/go-api-boilerplate/pkg/mysql"
)

// Token model
type Token struct {
	ID       string           `json:"id"`
	ClientID string           `json:"client_d,omitempty"`
	UserID   string           `json:"user_id,omitempty"`
	Scope    string           `json:"scope"`
	Access   string           `json:"access_token"`
	Refresh  string           `json:"refresh_token"`
	Code     mysql.NullString `json:"code"`
}

// GetID the id
func (t Token) GetID() string {
	return t.ID
}

// GetClientID the client id
func (t Token) GetClientID() string {
	return t.ClientID
}

// GetUserID the user id
func (t Token) GetUserID() string {
	return t.UserID
}

// GetAccess access Token
func (t Token) GetAccess() string {
	return t.Access
}

// GetRefresh refresh Token
func (t Token) GetRefresh() string {
	return t.Refresh
}

// GetScope get scope of authorization
func (t Token) GetScope() string {
	return t.Scope
}

// GetCode authorization code
func (t Token) GetCode() string {
	return t.Code.String
}
