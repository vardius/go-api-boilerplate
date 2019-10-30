package oauth2

import (
	"fmt"

	_ "github.com/go-sql-driver/mysql"
	"github.com/vardius/go-api-boilerplate/cmd/user/internal/application/config"
	"golang.org/x/oauth2"
)

// NewConfig provides oauth2 config
func NewConfig() oauth2.Config {
	return oauth2.Config{
		ClientID:     config.Env.ClientID,
		ClientSecret: config.Env.ClientSecret,
		Scopes:       []string{"all"},
		Endpoint: oauth2.Endpoint{
			AuthURL:  fmt.Sprintf("http://%s:%d/authorize", config.Env.AuthHost, config.Env.PortHTTP),
			TokenURL: fmt.Sprintf("http://%s:%d/token", config.Env.AuthHost, config.Env.PortHTTP),
		},
	}
}
