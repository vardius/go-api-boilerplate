/*
Package mysql holds view model repositories
*/
package mysql

// Client model
type Client struct {
	ID          string   `json:"id"`
	UserID      string   `json:"user_id"`
	Secret      string   `json:"secret"`
	Domain      string   `json:"domain"`
	RedirectURL string   `json:"redirect_url"`
	Scopes      []string `json:"scopes"`
}

// GetID client id
func (c Client) GetID() string {
	return c.ID
}

// GetSecret client domain
func (c Client) GetSecret() string {
	return c.Secret
}

// GetDomain client domain
func (c Client) GetDomain() string {
	return c.Domain
}

// GetUserID user id
func (c Client) GetUserID() string {
	return c.UserID
}

// RedirectURL user id
func (c Client) GetRedirectURL() string {
	return c.RedirectURL
}

// Scopes user id
func (c Client) GetScopes() []string {
	return c.Scopes
}
