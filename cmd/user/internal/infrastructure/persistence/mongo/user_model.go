package mongo

import (
	"github.com/vardius/go-api-boilerplate/cmd/user/internal/application/access"
)

// User model
type User struct {
	ID         string      `json:"id" bson:"user_id"`
	Email      string      `json:"email" bson:"email_address"`
	FacebookID string      `json:"facebook_id" bson:"facebook_id,omitempty"`
	GoogleID   string      `json:"google_id" bson:"google_id,omitempty"`
	Role       access.Role `json:"role" bson:"role"`
}

// GetID the id
func (u User) GetID() string {
	return u.ID
}

// GetEmail the email
func (u User) GetEmail() string {
	return u.Email
}

// GetFacebookID facebook id
func (u User) GetFacebookID() string {
	return u.FacebookID
}

// GetGoogleID google id
func (u User) GetGoogleID() string {
	return u.GoogleID
}

// GetRole returns user role
func (u User) GetRole() access.Role {
	return u.Role
}
