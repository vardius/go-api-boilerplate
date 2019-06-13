/*
Package persistence holds view models and repository interfaces
*/
package persistence

import (
	"context"
)

// User model
type User struct {
	ID         string  `json:"id"`
	Email      string  `json:"emailAddress"`
	FacebookID *string `json:"facebookId"`
	GoogleID   *string `json:"googleId"`
}

// UserRepository allows to get/save current state of user to mysql storage
type UserRepository interface {
	FindAll(ctx context.Context, limit, offset int32) ([]*User, error)
	Get(ctx context.Context, id string) (*User, error)
	Add(ctx context.Context, user *User) error
	Delete(ctx context.Context, id string) error
	Count(ctx context.Context) (int32, error)
	UpdateEmail(ctx context.Context, id, email string) error
	UpdateFacebookID(ctx context.Context, id, facebookID string) error
	UpdateGoogleID(ctx context.Context, id, googleID string) error
}
