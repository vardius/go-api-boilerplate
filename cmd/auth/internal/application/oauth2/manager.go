package oauth2

import (
	"github.com/dgrijalva/jwt-go"
	"gopkg.in/oauth2.v3"
	oauth2_generates "gopkg.in/oauth2.v3/generates"
	oauth2_manage "gopkg.in/oauth2.v3/manage"
)

// NewManager initialize the oauth2 manager service
func NewManager(tokenStore oauth2.TokenStore, clientStore oauth2.ClientStore, secretKey []byte) oauth2.Manager {
	manager := oauth2_manage.NewDefaultManager()

	manager.SetPasswordTokenCfg(oauth2_manage.DefaultPasswordTokenCfg)
	manager.MapTokenStorage(tokenStore)
	manager.MapClientStorage(clientStore)
	manager.MapAccessGenerate(oauth2_generates.NewJWTAccessGenerate(secretKey, jwt.SigningMethodHS512))

	return manager
}
