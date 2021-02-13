/*
Package persistence holds view models and repository interfaces
*/
package persistence

import (
	"context"

	"github.com/vardius/go-api-boilerplate/cmd/user/internal/application/access"
)

// User persistence model interface
type User interface {
	GetID() string
	GetEmail() string
	GetFacebookID() string
	GetGoogleID() string
	GetRole() access.Role
}

// UserRepository allows to get/save user to mysql storage
type UserRepository interface {
	FindAll(ctx context.Context, limit, offset int64) ([]User, error)
	Get(ctx context.Context, id string) (User, error)
	GetByEmail(ctx context.Context, email string) (User, error)
	GetByFacebookID(ctx context.Context, facebookID string) (User, error)
	GetByGoogleID(ctx context.Context, googleID string) (User, error)
	Add(ctx context.Context, user User) error
	Delete(ctx context.Context, id string) error
	Count(ctx context.Context) (int64, error)
	UpdateEmail(ctx context.Context, id, email string) error
	UpdateFacebookID(ctx context.Context, id, facebookID string) error
	UpdateGoogleID(ctx context.Context, id, googleID string) error
}
