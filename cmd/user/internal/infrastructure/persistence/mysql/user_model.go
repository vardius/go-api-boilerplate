/*
Package mysql holds view model repositories
*/
package mysql

import "github.com/vardius/go-api-boilerplate/internal/mysql"

// User model
type User struct {
	ID           string           `json:"id"`
	Provider     mysql.NullString `json:"provider"`
	Name         mysql.NullString `json:"name"`
	Email        string           `json:"email"`
	NickName     mysql.NullString `json:"nickName"`
	Location     mysql.NullString `json:"location"`
	AvatarURL    mysql.NullString `json:"avatarURL"`
	Description  mysql.NullString `json:"description"`
	UserID       mysql.NullString `json:"userId"`
	RefreshToken mysql.NullString `json:"refreshToken"`
}

// GetID the id
func (u User) GetID() string {
	return u.ID
}

// GetProvider the provider
func (u User) GetProvider() string {
	return u.Provider.String
}

// GetProvider the provider
func (u User) GetName() string {
	return u.Name.String
}

// GetEmail the email
func (u User) GetEmail() string {
	return u.Email
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
