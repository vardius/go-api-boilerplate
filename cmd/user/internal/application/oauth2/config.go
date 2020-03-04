package oauth2

import (
	"fmt"

	"golang.org/x/oauth2"

	"github.com/vardius/go-api-boilerplate/cmd/user/internal/application/config"
)

// NewConfigFacebook provides oauth2 config
func NewConfigFacebook() oauth2.Config {
	return oauth2.Config{
		ClientID:     config.Env.Facebook.ClientID,
		ClientSecret: config.Env.Facebook.ClientSecret,
		Scopes:       []string{"all"},
		RedirectURL:  fmt.Sprintf("http://%s:%d/v1/facebook/callback", config.Env.User.Host, config.Env.HTTP.Port),
	}
}

// NewConfigGoogle provides oauth2 config
func NewConfigGoogle() oauth2.Config {
	return oauth2.Config{
		ClientID:     config.Env.Google.ClientID,
		ClientSecret: config.Env.Google.ClientSecret,
		Scopes:       []string{"all"},
		RedirectURL:  fmt.Sprintf("http://%s:%d/v1/google/callback", config.Env.User.Host, config.Env.HTTP.Port),
	}
}
