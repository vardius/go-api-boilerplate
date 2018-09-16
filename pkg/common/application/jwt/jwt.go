/*
Package jwt allows to encode/decode identity to jwt tokens
*/
package jwt

import (
	"errors"
	"fmt"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/vardius/go-api-boilerplate/pkg/common/application/identity"
)

// Jwt allows to encode/decode identity to jwt tokens
type Jwt interface {
	Encode(*identity.Identity) (string, error)
	Decode(token string) (*identity.Identity, error)
}

// IdentityClaims are encoded as a JSON object that is digitally signed and optionally encrypted.
// for more claims see: https://godoc.org/github.com/dgrijalva/jwt-go#StandardClaims
type IdentityClaims struct {
	Identity *identity.Identity `json:"identity,omitempty"`
	jwt.StandardClaims
}

type jwtService struct {
	signingKey []byte
	expiration time.Duration
}

// Encode creates JWT token with encoded identity
func (s *jwtService) Encode(i *identity.Identity) (string, error) {
	claims := IdentityClaims{
		i,
		jwt.StandardClaims{
			ExpiresAt: time.Now().Add(s.expiration).Unix(),
			Issuer:    "test",
		},
	}

	// https://godoc.org/github.com/dgrijalva/jwt-go#NewWithClaims
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	tokenString, err := token.SignedString(s.signingKey)
	if err != nil {
		return "", errors.New("Failed to sign token")
	}

	return tokenString, nil
}

// Decode decodes given token and returns identity, implements TokenAuthFunc type
func (s *jwtService) Decode(token string) (*identity.Identity, error) {
	parsedToken, err := jwt.ParseWithClaims(token, &IdentityClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
		}
		return s.signingKey, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := parsedToken.Claims.(*IdentityClaims); ok && parsedToken.Valid {
		return claims.Identity, nil
	}

	return nil, errors.New("Error parsing token")
}

// New return new instance of Jwt
func New(signingKey []byte, expiration time.Duration) Jwt {
	return &jwtService{signingKey, expiration}
}
