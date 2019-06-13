/*
Package mysql holds view model repositories
*/
package mysql

import (
	"encoding/json"

	oauth2 "gopkg.in/oauth2.v3"
)

// Client model
type Client struct {
	ID     string          `json:"id"`
	UserID string          `json:"userId"`
	Secret string          `json:"secret"`
	Domain string          `json:"domain"`
	Data   json.RawMessage `json:"data"`
}

// GetInfo client oauth2 info
func (c *Client) GetInfo() (i oauth2.ClientInfo) {
	json.Unmarshal(c.Data, &i)

	return i
}

// GetID client id
func (c *Client) GetID() string {
	return c.ID
}

// GetSecret client domain
func (c *Client) GetSecret() string {
	return c.Secret
}

// GetDomain client domain
func (c *Client) GetDomain() string {
	return c.Domain
}

// GetUserID user id
func (c *Client) GetUserID() string {
	return c.UserID
}

// GetData client data
func (c *Client) GetData() json.RawMessage {
	return c.Data
}
