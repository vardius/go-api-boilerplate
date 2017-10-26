package auth

import (
	"errors"
	"fmt"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
)

const identityClaimKey = "identity"

// JwtService allows to encode/decode jwt tokens
type JwtService interface {
	GenerateToken(*Identity) (string, error)
	Authenticate(token string) (*Identity, error)
}

type jwtService struct {
	signinKey  []byte
	expiration time.Duration
}

func (s *jwtService) GenerateToken(i *Identity) (string, error) {
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

func (s *jwtService) Authenticate(token string) (*Identity, error) {
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
		return claims[identityClaimKey].(*Identity), nil
	}

	return nil, errors.New("Error parsing token")
}

// NewJwtService return new instance of JwtService
func NewJwtService(signinKey []byte, expiration time.Duration) JwtService {
	return &jwtService{signinKey, expiration}
}
