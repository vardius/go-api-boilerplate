/*
Package mysql holds view model repositories
*/
package mysql

// User model
type User struct {
	ID    string `json:"id"`
	Email string `json:"email"`
}

// GetID the id
func (u User) GetID() string {
	return u.ID
}

// GetEmail the email
func (u User) GetEmail() string {
	return u.Email
}
