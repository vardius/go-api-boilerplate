package auth

import (
	"fmt"

	"github.com/dgrijalva/jwt-go"

	"github.com/vardius/go-api-boilerplate/pkg/identity"
)

type Claims struct {
	jwt.StandardClaims
	Identity identity.Identity `json:"identity"`
}

func (c *Claims) Valid() error {
	if c.Identity.ID.ID() == 0 {
		return fmt.Errorf("user ID must be set")
	}
	if c.Identity.Email == "" {
		return fmt.Errorf("user email must be set")
	}

	return c.StandardClaims.Valid()
}
