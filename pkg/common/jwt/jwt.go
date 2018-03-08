/*
Package jwt allows to encode/decode identity to jwt tokens
*/
package jwt

import (
	"errors"
	"fmt"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/vardius/go-api-boilerplate/pkg/common/security/identity"
)

const identityClaimKey = "identity"

// Jwt allows to encode/decode identity to jwt tokens
type Jwt interface {
	Encode(*identity.Identity) (string, error)
	Decode(token string) (*identity.Identity, error)
}

type jwtService struct {
	signinKey  []byte
	expiration time.Duration
}

// Encode creates JWT token with encoded identity
func (s *jwtService) Encode(i *identity.Identity) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		identityClaimKey: i,
		"nbf":            time.Now().Add(s.expiration).Unix(),
	})

	tokenString, err := token.SignedString(s.signinKey)
	if err != nil {
		return "", errors.New("Failed to sign token")
	}

	return tokenString, nil
}

// Decode decodes given token and returns identity, imlpements TokenAuthFunc type
func (s *jwtService) Decode(token string) (*identity.Identity, error) {
	parsedToken, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
		}
		return s.signinKey, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := parsedToken.Claims.(jwt.MapClaims); ok && parsedToken.Valid {
		return claims[identityClaimKey].(*identity.Identity), nil
	}

	return nil, errors.New("Error parsing token")
}

// New return new instance of Jwt
func New(signinKey []byte, expiration time.Duration) Jwt {
	return &jwtService{signinKey, expiration}
}
