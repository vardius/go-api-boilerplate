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

	"github.com/vardius/go-api-boilerplate/pkg/auth"
	apperrors "github.com/vardius/go-api-boilerplate/pkg/errors"
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
func NewJWTAccess(method jwt.SigningMethod, authenticator auth.Authenticator) *JWTAccess {
	return &JWTAccess{
		signedMethod:  method,
		authenticator: authenticator,
	}
}

// JWTAccess generate the jwt access token
type JWTAccess struct {
	signedMethod  jwt.SigningMethod
	authenticator auth.Authenticator
}

// Token based on the UUID generated token
func (a *JWTAccess) Token(ctx context.Context, data *oauth2.GenerateBasic, isGenRefresh bool) (string, string, error) {
	var userID id.UUID
	if data.UserID != "" {
		userUUID, err := id.Parse(data.UserID)
		if err != nil {
			return "", "", apperrors.Wrap(err)
		}
		userID = userUUID
	}

	clientID, err := id.Parse(data.Client.GetID())
	if err != nil {
		return "", "", apperrors.Wrap(err)
	}

	var expiresAt int64
	if data.TokenInfo.GetAccessExpiresIn() != 0 {
		expiresAt = data.TokenInfo.GetAccessCreateAt().Add(data.TokenInfo.GetAccessExpiresIn()).Unix()
	}

	claims := &JWTAccessClaims{
		Claims: auth.Claims{
			StandardClaims: jwt.StandardClaims{
				Audience:  data.Client.GetID(),
				Subject:   data.UserID,
				ExpiresAt: expiresAt,
			},
			UserID:   userID,
			ClientID: clientID,
		},
	}

	token := jwt.NewWithClaims(a.signedMethod, claims)

	access, err := a.authenticator.Sign(token)
	if err != nil {
		return "", "", apperrors.Wrap(err)
	}
	refresh := ""

	if isGenRefresh {
		refresh = base64.URLEncoding.EncodeToString(uuid.NewSHA1(uuid.Must(uuid.NewRandom()), []byte(access)).Bytes())
		refresh = strings.ToUpper(strings.TrimRight(refresh, "="))
	}

	return access, refresh, nil
}
