package auth

import (
	"fmt"

	"github.com/dgrijalva/jwt-go"

	"github.com/vardius/go-api-boilerplate/pkg/application"
	"github.com/vardius/go-api-boilerplate/pkg/errors"
	"github.com/vardius/go-api-boilerplate/pkg/identity"
)

type TokenAuthorizer interface {
	Auth(token string) (identity.Identity, error)
}

type jwtAuthorizer struct {
	claimsProvider ClaimsProvider
}

func NewJWTTokenAuthorizer(claimsProvider ClaimsProvider) TokenAuthorizer {
	return &jwtAuthorizer{
		claimsProvider: claimsProvider,
	}
}

func (a *jwtAuthorizer) Auth(token string) (identity.Identity, error) {
	c, err := a.claimsProvider.FromJWT(token)
	if err != nil {
		ve, ok := err.(*jwt.ValidationError)
		if ok {
			switch {
			case ve.Errors&jwt.ValidationErrorMalformed != 0:
				err = errors.Wrap(fmt.Errorf("token is malformed: %w", err))
			case ve.Errors&jwt.ValidationErrorUnverifiable != 0:
				err = errors.Wrap(fmt.Errorf("token could not be verified because of signing problems: %w", err))
			case ve.Errors&jwt.ValidationErrorSignatureInvalid != 0:
				err = errors.Wrap(fmt.Errorf("signature validation failed: %w", err))

			// Standard Claim validation errors
			case ve.Errors&jwt.ValidationErrorAudience != 0:
				err = errors.Wrap(fmt.Errorf("AUD validation failed: %w", err))
			case ve.Errors&jwt.ValidationErrorExpired != 0:
				err = errors.Wrap(fmt.Errorf("EXP validation failed: %w", err))
			case ve.Errors&jwt.ValidationErrorIssuedAt != 0:
				err = errors.Wrap(fmt.Errorf("IAT validation failed: %w", err))
			case ve.Errors&jwt.ValidationErrorIssuer != 0:
				err = errors.Wrap(fmt.Errorf("ISS validation failed: %w", err))
			case ve.Errors&jwt.ValidationErrorNotValidYet != 0:
				err = errors.Wrap(fmt.Errorf("NBF validation failed: %w", err))
			case ve.Errors&jwt.ValidationErrorId != 0:
				err = errors.Wrap(fmt.Errorf("JTI validation failed: %w", err))
			case ve.Errors&jwt.ValidationErrorClaimsInvalid != 0:
				err = errors.Wrap(fmt.Errorf("generic claims validation error: %w", err))
			}
		}

		return identity.NullIdentity, errors.Wrap(fmt.Errorf("%w: Could not verify token: %s", application.ErrUnauthorized, err))
	}

	return c.Identity.WithToken(token), nil
}
