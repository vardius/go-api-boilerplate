/*
Package mysql holds view model repositories
*/
package mysql

import (
	"encoding/json"
)

// Client model
type Client struct {
	ID     string          `json:"id"`
	UserID string          `json:"user_id"`
	Secret string          `json:"secret"`
	Domain string          `json:"domain"`
	Data   json.RawMessage `json:"data"`
}

// GetID client id
func (c Client) GetID() string {
	return c.ID
}

// GetSecret client domain
func (c Client) GetSecret() string {
	return c.Secret
}

// GetDomain client domain
func (c Client) GetDomain() string {
	return c.Domain
}

// GetUserID user id
func (c Client) GetUserID() string {
	return c.UserID
}

// GetData client data
func (c Client) GetData() json.RawMessage {
	return c.Data
}
