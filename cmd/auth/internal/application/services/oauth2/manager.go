package oauth2

import (
	"github.com/dgrijalva/jwt-go"
	"gopkg.in/oauth2.v4"
	oauth2manage "gopkg.in/oauth2.v4/manage"

	"github.com/vardius/go-api-boilerplate/cmd/auth/internal/infrastructure/persistence"
	"github.com/vardius/go-api-boilerplate/pkg/auth"
)

// NewManager initialize the oauth2 manager service
func NewManager(tokenStore oauth2.TokenStore, clientStore oauth2.ClientStore, authenticator auth.Authenticator, clientRepository persistence.ClientRepository) oauth2.Manager {
	manager := oauth2manage.NewDefaultManager()

	manager.SetAuthorizeCodeTokenCfg(oauth2manage.DefaultAuthorizeCodeTokenCfg)
	manager.SetClientTokenCfg(oauth2manage.DefaultClientTokenCfg)
	manager.SetAuthorizeCodeTokenCfg(oauth2manage.DefaultAuthorizeCodeTokenCfg)
	manager.SetRefreshTokenCfg(oauth2manage.DefaultRefreshTokenCfg)
	manager.MapTokenStorage(tokenStore)
	manager.MapClientStorage(clientStore)
	manager.MapAccessGenerate(NewJWTAccess(jwt.SigningMethodHS512, authenticator, clientRepository))

	return manager
}
