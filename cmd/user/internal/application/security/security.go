package application

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	authproto "github.com/vardius/go-api-boilerplate/cmd/auth/proto"
	userpersistence "github.com/vardius/go-api-boilerplate/cmd/user/internal/infrastructure/persistence"
	"github.com/vardius/go-api-boilerplate/pkg/application"
	"github.com/vardius/go-api-boilerplate/pkg/auth"
	"github.com/vardius/go-api-boilerplate/pkg/errors"
	httpauthenticator "github.com/vardius/go-api-boilerplate/pkg/http/middleware/authenticator"
	"github.com/vardius/go-api-boilerplate/pkg/identity"
)

// TokenAuthSecretHandler provides token auth function
// will verify token against service secret key
func TokenAuthSecretHandler(provider auth.ClaimsProvider) httpauthenticator.TokenAuthFunc {
	fn := func(token string) (identity.Identity, error) {
		c, err := provider.FromJWT(token)
		if err != nil {
			return identity.NullIdentity, errors.Wrap(fmt.Errorf("%w: Could not verify token: %s", application.ErrUnauthorized, err))
		}

		return c.Identity.WithToken(token), nil
	}

	return fn
}

// TokenAuthOauthHandler provides token auth function
// will send gRPC request to auth service to verify token
func TokenAuthOauthHandler(grpAuthClient authproto.AuthenticationServiceClient, repository userpersistence.UserRepository, timeout time.Duration) httpauthenticator.TokenAuthFunc {
	fn := func(token string) (identity.Identity, error) {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		tokenInfo, err := grpAuthClient.VerifyToken(ctx, &authproto.VerifyTokenRequest{
			Token: token,
		})
		if err != nil {
			return identity.NullIdentity, errors.Wrap(fmt.Errorf("%w: Could not verify token: %s", application.ErrUnauthorized, err))
		}

		user, err := repository.Get(ctx, tokenInfo.GetUserId())
		if err != nil {
			return identity.NullIdentity, errors.Wrap(err)
		}

		userID, err := uuid.Parse(user.GetID())
		if err != nil {
			return identity.NullIdentity, errors.Wrap(err)
		}

		i := identity.Identity{
			ID:    userID,
			Token: token,
			Email: user.GetEmail(),
			Roles: identity.RoleUser,
		}

		return i, nil
	}

	return fn
}
