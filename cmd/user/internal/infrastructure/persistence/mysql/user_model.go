/*
Package mysql holds view model repositories
*/
package mysql

import (
	"github.com/vardius/go-api-boilerplate/pkg/mysql"
)

// User model
type User struct {
	ID         string           `json:"id"`
	Email      string           `json:"email"`
	FacebookID mysql.NullString `json:"facebook_id"`
	GoogleID   mysql.NullString `json:"google_id"`
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
	return u.FacebookID.String
}

// GetGoogleID google id
func (u User) GetGoogleID() string {
	return u.GoogleID.String
}
