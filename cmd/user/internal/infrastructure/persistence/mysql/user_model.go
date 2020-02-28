/*
Package mysql holds view model repositories
*/
package mysql

import (
	"github.com/vardius/go-api-boilerplate/internal/mysql"
)

// User model
type User struct {
	ID         string           `json:"id"`
	Name       string           `json:"name"`
	Email      string           `json:"emailAddress"`
	Password   string           `json:"password"`
	FacebookID mysql.NullString `json:"facebookId"`
	GoogleID   mysql.NullString `json:"googleId"`
}

// GetID the id
func (u User) GetID() string {
	return u.ID
}

// Get full name
func (u User) GetName() string {
	return u.Name
}

// GetEmail the email
func (u User) GetEmail() string {
	return u.Email
}

// Get password
func (u User) GetPassword() string {
	return u.Password
}

// GetFacebookID facebook id
func (u User) GetFacebookID() string {
	return u.FacebookID.String
}

// GetGoogleID google id
func (u User) GetGoogleID() string {
	return u.GoogleID.String
}
