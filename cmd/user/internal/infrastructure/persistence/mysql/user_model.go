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
	Email      string           `json:"emailAddress"`
	FacebookID mysql.NullString `json:"facebookId"`
	GoogleID   mysql.NullString `json:"googleId"`
}

// GetID the id
func (u User) GetID() string {
	return u.ID
}

// GetEmail the email
func (u User) GetEmail() string {
	return u.Email
}

// GetPassword the password
func (u User) GetPassword() string {
	return u.Password.String
}

// GetNickName the nickname
func (u User) GetNickName() string {
	return u.NickName.String
}

// GetLocation the location
func (u User) GetLocation() string {
	return u.Location.String
}

// GetAvatarURL the avatarurl
func (u User) GetAvatarURL() string {
	return u.AvatarURL.String
}

// GetDescription the description
func (u User) GetDescription() string {
	return u.Description.String
}

// GetUserID the userid
func (u User) GetUserID() string {
	return u.UserID.String
}

// GetRefreshToken the refreshtoken
func (u User) GetRefreshToken() string {
	return u.RefreshToken.String
}
