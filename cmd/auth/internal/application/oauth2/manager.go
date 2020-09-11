package oauth2

import (
	"time"

	"github.com/dgrijalva/jwt-go"
	"gopkg.in/oauth2.v4"
	oauth2manage "gopkg.in/oauth2.v4/manage"

	"github.com/vardius/go-api-boilerplate/pkg/auth"
)

var (
	PasswordTokenCfg = &oauth2manage.Config{
		AccessTokenExp:    0, // access token expiration time, 0 means it doesn't expire
		RefreshTokenExp:   time.Hour * 24 * 7,
		IsGenerateRefresh: false,
	}
)

// NewManager initialize the oauth2 manager service
func NewManager(tokenStore oauth2.TokenStore, clientStore oauth2.ClientStore, authenticator auth.Authenticator) oauth2.Manager {
	manager := oauth2manage.NewDefaultManager()

	manager.SetAuthorizeCodeTokenCfg(oauth2manage.DefaultAuthorizeCodeTokenCfg)
	manager.SetClientTokenCfg(oauth2manage.DefaultClientTokenCfg)
	manager.SetAuthorizeCodeTokenCfg(oauth2manage.DefaultAuthorizeCodeTokenCfg)
	manager.SetRefreshTokenCfg(oauth2manage.DefaultRefreshTokenCfg)
	manager.SetPasswordTokenCfg(PasswordTokenCfg)
	manager.MapTokenStorage(tokenStore)
	manager.MapClientStorage(clientStore)
	manager.MapAccessGenerate(NewJWTAccess(jwt.SigningMethodHS512, authenticator))

	return manager
}
