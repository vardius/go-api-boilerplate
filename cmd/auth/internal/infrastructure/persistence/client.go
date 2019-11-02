/*
Package persistence holds view models and repository interfaces
*/
package persistence

import (
	"context"
	"encoding/json"
)

// Client the client persistence model interface
type Client interface {
	GetID() string
	GetUserID() string
	GetSecret() string
	GetDomain() string
	GetData() json.RawMessage
}

// ClientRepository allows to get/save current state of user to mysql storage
type ClientRepository interface {
	Get(ctx context.Context, id string) (Client, error)
	Add(ctx context.Context, client Client) error
	Delete(ctx context.Context, id string) error
}
