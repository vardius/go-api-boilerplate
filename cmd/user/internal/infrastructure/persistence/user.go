/*
Package persistence holds view models and repository interfaces
*/
package persistence

import (
	"context"
)

// User the user persistence model interface
type User interface {
	GetID() string
	GetEmail() string
	GetFacebookID() string
	GetGoogleID() string
}

// UserRepository allows to get/save current state of user to mysql storage
type UserRepository interface {
	FindAll(ctx context.Context, limit, offset int32) ([]User, error)
	Get(ctx context.Context, id string) (User, error)
	GetByEmail(ctx context.Context, email string) (User, error)
	Add(ctx context.Context, user User) error
	Delete(ctx context.Context, id string) error
	Count(ctx context.Context) (int32, error)
	UpdateEmail(ctx context.Context, id, email string) error
	UpdateFacebookID(ctx context.Context, id, facebookID string) error
	UpdateGoogleID(ctx context.Context, id, googleID string) error
}
