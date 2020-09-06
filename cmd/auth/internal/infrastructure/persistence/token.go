/*
Package persistence holds view models and repository interfaces
*/
package persistence

import (
	"context"
)

// Token the token persistence model interface
type Token interface {
	GetID() string
	GetClientID() string
	GetUserID() string
	GetAccess() string
	GetRefresh() string
	GetScope() string
	GetCode() string
}

// TokenRepository allows to get/save current state of token to mysql storage
type TokenRepository interface {
	Get(ctx context.Context, id string) (Token, error)
	GetByCode(ctx context.Context, code string) (Token, error)
	GetByAccess(ctx context.Context, access string) (Token, error)
	GetByRefresh(ctx context.Context, refresh string) (Token, error)
	Add(ctx context.Context, token Token) error
	Delete(ctx context.Context, id string) error

	CountByUserID(ctx context.Context, clientID, userID string) (int32, error)
	FindAllByUserID(ctx context.Context, clientID, userID string, limit, offset int32) ([]Token, error)
}
