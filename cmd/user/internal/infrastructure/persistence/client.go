/*
Package persistence holds view models and repository interfaces
*/
package persistence

import (
	"context"
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

// ClientRepository allows to get current client from mysql storage
type ClientRepository interface {
	Get(ctx context.Context, id string) (Client, error)
	GetByUserDomain(ctx context.Context, userID, domain string) (Client, error)
}
