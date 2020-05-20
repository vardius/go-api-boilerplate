package oauth2

import (
	"github.com/dgrijalva/jwt-go"
	"gopkg.in/oauth2.v3"
	oauth2generates "gopkg.in/oauth2.v3/generates"
	oauth2manage "gopkg.in/oauth2.v3/manage"
)

// NewManager initialize the oauth2 manager service
func NewManager(tokenStore oauth2.TokenStore, clientStore oauth2.ClientStore, secretKey []byte) oauth2.Manager {
	manager := oauth2manage.NewDefaultManager()

	manager.SetPasswordTokenCfg(oauth2manage.DefaultPasswordTokenCfg)
	manager.MapTokenStorage(tokenStore)
	manager.MapClientStorage(clientStore)
	manager.MapAccessGenerate(oauth2generates.NewJWTAccessGenerate(secretKey, jwt.SigningMethodHS512))

	return manager
}
