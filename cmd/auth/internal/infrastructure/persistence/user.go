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
}

// UserRepository allows to get/save current state of user to mysql storage
type UserRepository interface {
	Get(ctx context.Context, id string) (User, error)
	GetByEmail(ctx context.Context, email string) (User, error)
}
