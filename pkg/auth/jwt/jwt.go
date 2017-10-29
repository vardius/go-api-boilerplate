package jwt

import (
	"app/pkg/auth/identity"
	"errors"
	"fmt"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
)

const identityClaimKey = "identity"

// Jwt allows to encode/decode jwt tokens with identity
type Jwt interface {
	GenerateToken(*identity.Identity) (string, error)
	Authenticate(token string) (*identity.Identity, error)
}

type jwtService struct {
	signinKey  []byte
	expiration time.Duration
}

func (s *jwtService) GenerateToken(i *identity.Identity) (string, error) {
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

func (s *jwtService) Authenticate(token string) (*identity.Identity, error) {
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
