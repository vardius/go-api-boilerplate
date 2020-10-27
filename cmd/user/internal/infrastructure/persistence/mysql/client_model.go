/*
Package mysql holds view model repositories
*/
package mysql

import (
	"gopkg.in/oauth2.v4"
	"gopkg.in/oauth2.v4/models"
)

// Client model
type Client struct {
	ID          string   `json:"id"`
	UserID      string   `json:"user_id"`
	Secret      string   `json:"secret"`
	Domain      string   `json:"domain"`
	RedirectURL string   `json:"redirect_url"`
	Scopes      []string `json:"scopes"`
}

func (c *Client) GetID() string {
	return c.ID
}

func (c *Client) GetUserID() string {
	return c.UserID
}

func (c *Client) GetSecret() string {
	return c.Secret
}

func (c *Client) GetDomain() string {
	return c.Domain
}

func (c *Client) GetRedirectURL() string {
	return c.RedirectURL
}

func (c *Client) GetScopes() []string {
	return c.Scopes
}

func (c *Client) ClientInfo() (oauth2.ClientInfo, error) {
	return &models.Client{
		ID:     c.ID,
		Secret: c.Secret,
		Domain: c.Domain,
		UserID: c.UserID,
	}, nil
}
