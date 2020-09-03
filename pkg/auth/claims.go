package auth

import (
	"fmt"

	"github.com/dgrijalva/jwt-go"
	"github.com/google/uuid"

	"github.com/vardius/go-api-boilerplate/pkg/errors"
)

type Claims struct {
	jwt.StandardClaims
	UserID   uuid.UUID `json:"user_id"`
	ClientID uuid.UUID `json:"client_id"`
}

func (c *Claims) Valid() error {
	if c.UserID.String() == "" {
		return errors.Wrap(fmt.Errorf("UserID must be set"))
	}
	if c.ClientID.String() == "" {
		return errors.Wrap(fmt.Errorf("ClientID must be set"))
	}

	return c.StandardClaims.Valid()
}
