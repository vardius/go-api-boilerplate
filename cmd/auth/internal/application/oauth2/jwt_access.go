package oauth2

import (
	"context"
	"encoding/base64"
	"strings"
	"time"

	"github.com/dgrijalva/jwt-go"
	id "github.com/google/uuid"
	"gopkg.in/oauth2.v4"
	oauth2errors "gopkg.in/oauth2.v4/errors"
	"gopkg.in/oauth2.v4/utils/uuid"

	userpersistence "github.com/vardius/go-api-boilerplate/cmd/auth/internal/infrastructure/persistence"
	"github.com/vardius/go-api-boilerplate/pkg/auth"
	"github.com/vardius/go-api-boilerplate/pkg/errors"
	"github.com/vardius/go-api-boilerplate/pkg/identity"
)

// JWTAccessClaims jwt claims
type JWTAccessClaims struct {
	auth.Claims
}

// Valid claims verification
func (c *JWTAccessClaims) Valid() error {
	if time.Unix(c.ExpiresAt, 0).Before(time.Now()) {
		return oauth2errors.ErrInvalidAccessToken
	}
	return c.Claims.Valid()
}

// NewJWTAccess create to generate the jwt access token instance
func NewJWTAccess(method jwt.SigningMethod, authenticator auth.Authenticator, repository userpersistence.UserRepository) *JWTAccess {
	return &JWTAccess{
		signedMethod:  method,
		authenticator: authenticator,
		repository:    repository,
	}
}

// JWTAccess generate the jwt access token
type JWTAccess struct {
	signedMethod  jwt.SigningMethod
	authenticator auth.Authenticator
	repository    userpersistence.UserRepository
}

// Token based on the UUID generated token
func (a *JWTAccess) Token(ctx context.Context, data *oauth2.GenerateBasic, isGenRefresh bool) (string, string, error) {
	user, err := a.repository.Get(ctx, data.TokenInfo.GetUserID()) // @TODO: call user service to get user info
	if err != nil {
		return "", "", errors.Wrap(err)
	}

	userID, err := id.Parse(user.GetID())
	if err != nil {
		return "", "", errors.Wrap(err)
	}

	claims := &JWTAccessClaims{
		Claims: auth.Claims{
			StandardClaims: jwt.StandardClaims{
				Audience:  data.Client.GetID(),
				Subject:   data.UserID,
				ExpiresAt: data.TokenInfo.GetAccessCreateAt().Add(data.TokenInfo.GetAccessExpiresIn()).Unix(),
			},
			Identity: identity.Identity{
				ID:    userID,
				Token: data.TokenInfo.GetAccess(),
				Email: user.GetEmail(),
				Roles: identity.RoleUser,
			},
		},
	}

	token := jwt.NewWithClaims(a.signedMethod, claims)

	access, err := a.authenticator.Sign(token)
	if err != nil {
		return "", "", err
	}
	refresh := ""

	if isGenRefresh {
		refresh = base64.URLEncoding.EncodeToString(uuid.NewSHA1(uuid.Must(uuid.NewRandom()), []byte(access)).Bytes())
		refresh = strings.ToUpper(strings.TrimRight(refresh, "="))
	}

	return access, refresh, nil
}
