/*
Package mysql holds view model repositories
*/
package mysql

<<<<<<< HEAD
import (
	"github.com/vardius/go-api-boilerplate/internal/mysql"
)
=======
import "github.com/vardius/go-api-boilerplate/pkg/mysql"
>>>>>>> 7a0c2bb... move from internal packages to exported ones

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

// GetFacebookID facebook id
func (u User) GetFacebookID() string {
	return u.FacebookID.String
}

// GetGoogleID google id
func (u User) GetGoogleID() string {
	return u.GoogleID.String
}
