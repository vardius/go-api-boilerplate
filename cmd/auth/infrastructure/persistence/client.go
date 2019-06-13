/*
Package persistence holds view models and repository interfaces
*/
package persistence

import (
	"context"

	oauth2 "gopkg.in/oauth2.v3"
)

// Client model
type Client struct {
	ID     string
	UserID string
	Secret string
	Domain string
	Info   oauth2.ClientInfo
}

// ClientRepository allows to get/save current state of user to mysql storage
type ClientRepository interface {
	Get(ctx context.Context, id string) (*Client, error)
	Add(ctx context.Context, client *Client) error
	Delete(ctx context.Context, id string) error
}
