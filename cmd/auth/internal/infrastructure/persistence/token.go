/*
Package persistence holds view models and repository interfaces
*/
package persistence

import (
	"context"
	"encoding/json"

	"gopkg.in/oauth2.v4"
)

// Token the token persistence model interface
type Token interface {
	GetID() string
	GetUserAgent() string
	GetData() json.RawMessage
	TokenInfo() (oauth2.TokenInfo, error)
}

// TokenRepository allows to get/save current state of token to memory storage
type TokenRepository interface {
	Get(ctx context.Context, id string) (Token, error)
	GetByCode(ctx context.Context, code string) (Token, error)
	GetByAccess(ctx context.Context, access string) (Token, error)
	GetByRefresh(ctx context.Context, refresh string) (Token, error)
	Add(ctx context.Context, token Token) error
	Delete(ctx context.Context, id string) error

	CountByClientID(ctx context.Context, clientID string) (int64, error)
	FindAllByClientID(ctx context.Context, clientID string, limit, offset int64) ([]Token, error)
}
