package auth

import (
	"context"
	"fmt"

	"github.com/dgrijalva/jwt-go"
	"github.com/google/uuid"

	"github.com/vardius/go-api-boilerplate/cmd/auth/proto"
	apperrors "github.com/vardius/go-api-boilerplate/pkg/errors"
	"github.com/vardius/go-api-boilerplate/pkg/identity"
)

type TokenAuthorizer interface {
	Auth(ctx context.Context, token string) (*identity.Identity, error)
}

type IdentityProvider interface {
	GetByUserID(ctx context.Context, userID, clientID uuid.UUID) (*identity.Identity, error)
}

type jwtAuthorizer struct {
	claimsProvider     ClaimsProvider
	identityAuthorizer IdentityProvider
	authClient         proto.AuthenticationServiceClient
}

func NewJWTTokenAuthorizer(authClient proto.AuthenticationServiceClient, claimsProvider ClaimsProvider, identityAuthorizer IdentityProvider) TokenAuthorizer {
	return &jwtAuthorizer{
		claimsProvider:     claimsProvider,
		identityAuthorizer: identityAuthorizer,
		authClient:         authClient,
	}
}

func (a *jwtAuthorizer) Auth(ctx context.Context, token string) (*identity.Identity, error) {
	resp, err := a.authClient.ValidationBearerToken(ctx, &proto.ValidationBearerTokenRequest{
		Token: token,
	})
	if err != nil {
		return nil, apperrors.Wrap(err)
	}

	c, err := a.claimsProvider.FromJWT(token)
	if err != nil {
		ve, ok := err.(*jwt.ValidationError)
		if ok {
			switch {
			case ve.Errors&jwt.ValidationErrorMalformed != 0:
				err = apperrors.Wrap(fmt.Errorf("token is malformed: %w", err))
			case ve.Errors&jwt.ValidationErrorUnverifiable != 0:
				err = apperrors.Wrap(fmt.Errorf("token could not be verified because of signing problems: %w", err))
			case ve.Errors&jwt.ValidationErrorSignatureInvalid != 0:
				err = apperrors.Wrap(fmt.Errorf("signature validation failed: %w", err))

			// Standard Claim validation errors
			case ve.Errors&jwt.ValidationErrorAudience != 0:
				err = apperrors.Wrap(fmt.Errorf("AUD validation failed: %w", err))
			case ve.Errors&jwt.ValidationErrorExpired != 0:
				err = apperrors.Wrap(fmt.Errorf("EXP validation failed: %w", err))
			case ve.Errors&jwt.ValidationErrorIssuedAt != 0:
				err = apperrors.Wrap(fmt.Errorf("IAT validation failed: %w", err))
			case ve.Errors&jwt.ValidationErrorIssuer != 0:
				err = apperrors.Wrap(fmt.Errorf("ISS validation failed: %w", err))
			case ve.Errors&jwt.ValidationErrorNotValidYet != 0:
				err = apperrors.Wrap(fmt.Errorf("NBF validation failed: %w", err))
			case ve.Errors&jwt.ValidationErrorId != 0:
				err = apperrors.Wrap(fmt.Errorf("JTI validation failed: %w", err))
			case ve.Errors&jwt.ValidationErrorClaimsInvalid != 0:
				err = apperrors.Wrap(fmt.Errorf("generic claims validation error: %w", err))
			}
		}

		return nil, apperrors.Wrap(fmt.Errorf("could not verify token %s: %w", token, err))
	}

	if c.ClientID.String() != resp.ClientID {
		return nil, apperrors.Wrap(fmt.Errorf("could not verify token credentials clientID: %s != %s", token, err))
	}

	if c.UserID.String() != resp.UserID {
		return nil, apperrors.Wrap(fmt.Errorf("could not verify token credentials userID: %s != %s", token, err))
	}

	i, err := a.identityAuthorizer.GetByUserID(ctx, c.UserID, c.ClientID)
	if err != nil {
		return nil, apperrors.Wrap(err)
	}

	// @TODO: store role in users table
	i.WithRole(identity.RoleUser)
	i.WithToken(token)

	return i, nil
}
