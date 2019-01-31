/*
Package persistence holds view models and repository interfaces
*/
package persistence

import (
	"context"

	"github.com/vardius/go-api-boilerplate/cmd/user/infrastructure/proto"
)

// UserRepository allows to get/save current state of user to mysql storage
type UserRepository interface {
	FindAll(ctx context.Context, limit, offset int32) ([]*proto.User, error)
	Get(ctx context.Context, id string) (*proto.User, error)
	Add(ctx context.Context, user *proto.User) error
	Delete(ctx context.Context, id string) error
	Count(ctx context.Context) (int32, error)
	UpdateEmail(ctx context.Context, id, email string) error
	UpdateFacebookID(ctx context.Context, id, facebookID string) error
	UpdateGoogleID(ctx context.Context, id, googleID string) error
}
