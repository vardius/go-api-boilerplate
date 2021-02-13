/*
Package persistence holds view models and repository interfaces
*/
package persistence

import (
	"context"

	"gopkg.in/oauth2.v4"
)

// Client the client persistence model interface
type Client interface {
	GetID() string
	GetUserID() string
	GetSecret() string
	GetDomain() string
	GetRedirectURL() string
	GetScopes() []string
}

// ClientRepository allows to get/save current state of user to memory storage
type ClientRepository interface {
	Get(ctx context.Context, id string) (Client, error)
	Add(ctx context.Context, client Client) error
	Delete(ctx context.Context, id string) error

	// GetByID Calls Get method and returns oauth2 client info
	// Implements client store interface
	GetByID(ctx context.Context, id string) (oauth2.ClientInfo, error)

	CountByUserID(ctx context.Context, userID string) (int64, error)
	FindAllByUserID(ctx context.Context, userID string, limit, offset int64) ([]Client, error)
}
