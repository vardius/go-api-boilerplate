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

	"github.com/vardius/go-api-boilerplate/cmd/auth/internal/application/access"
	"github.com/vardius/go-api-boilerplate/cmd/auth/internal/infrastructure/persistence"
	"github.com/vardius/go-api-boilerplate/pkg/auth"
	apperrors "github.com/vardius/go-api-boilerplate/pkg/errors"
	"github.com/vardius/go-api-boilerplate/pkg/identity"
)

// JWTAccessClaims jwt claims
type JWTAccessClaims struct {
	auth.Claims
}

// Valid claims verification
func (c *JWTAccessClaims) Valid() error {
	if c.ExpiresAt != 0 && time.Unix(c.ExpiresAt, 0).Before(time.Now()) {
		return oauth2errors.ErrInvalidAccessToken
	}

	return c.Claims.Valid()
}

// NewJWTAccess create to generate the jwt access token instance
func NewJWTAccess(method jwt.SigningMethod, authenticator auth.Authenticator, clientRepository persistence.ClientRepository) *JWTAccess {
	return &JWTAccess{
		signedMethod:     method,
		authenticator:    authenticator,
		clientRepository: clientRepository,
	}
}

// JWTAccess generate the jwt access token
type JWTAccess struct {
	signedMethod     jwt.SigningMethod
	authenticator    auth.Authenticator
	clientRepository persistence.ClientRepository
}

// Token based on the UUID generated token
func (a *JWTAccess) Token(ctx context.Context, data *oauth2.GenerateBasic, isGenRefresh bool) (string, string, error) {
	userID, err := id.Parse(data.UserID)
	if err != nil {
		return "", "", apperrors.Wrap(err)
	}

	clientID, err := id.Parse(data.Client.GetID())
	if err != nil {
		return "", "", apperrors.Wrap(err)
	}

	c, err := a.clientRepository.Get(ctx, clientID.String())
	if err != nil {
		return "", "", apperrors.Wrap(err)
	}

	var expiresAt int64
	if data.TokenInfo.GetAccessExpiresIn() != 0 {
		expiresAt = data.TokenInfo.GetAccessCreateAt().Add(data.TokenInfo.GetAccessExpiresIn()).Unix()
	}

	var permissions identity.Permission
	for _, scope := range c.GetScopes() {
		switch access.Scope(scope) {
		case access.ScopeAll:
			permissions.Add(identity.PermissionUserRead)
			permissions.Add(identity.PermissionUserWrite)
			break
		case access.ScopeUserRead:
			permissions.Add(identity.PermissionUserRead)
		case access.ScopeUserWrite:
			permissions.Add(identity.PermissionUserWrite)
		}
	}

	claims := &JWTAccessClaims{
		Claims: auth.Claims{
			StandardClaims: jwt.StandardClaims{
				Audience:  data.Client.GetID(),
				Subject:   data.UserID,
				ExpiresAt: expiresAt,
			},
			Identity: &identity.Identity{
				Permission:   permissions,
				UserID:       userID,
				ClientID:     clientID,
				ClientDomain: c.GetDomain(),
			},
		},
	}

	token := jwt.NewWithClaims(a.signedMethod, claims)

	access, err := a.authenticator.Sign(token)
	if err != nil {
		return "", "", apperrors.Wrap(err)
	}

	var refresh string
	if isGenRefresh {
		refresh = base64.URLEncoding.EncodeToString(uuid.NewSHA1(uuid.Must(uuid.NewRandom()), []byte(access)).Bytes())
		refresh = strings.ToUpper(strings.TrimRight(refresh, "="))
	}

	return access, refresh, nil
}
