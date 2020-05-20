package application

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	authproto "github.com/vardius/go-api-boilerplate/cmd/auth/proto"
	userpersistence "github.com/vardius/go-api-boilerplate/cmd/user/internal/infrastructure/persistence"
	"github.com/vardius/go-api-boilerplate/pkg/application"
	"github.com/vardius/go-api-boilerplate/pkg/errors"
	httpauthenticator "github.com/vardius/go-api-boilerplate/pkg/http/middleware/authenticator"
	"github.com/vardius/go-api-boilerplate/pkg/identity"
)

// TokenAuthHandler provides token auth function
func TokenAuthHandler(grpAuthClient authproto.AuthenticationServiceClient, repository userpersistence.UserRepository) httpauthenticator.TokenAuthFunc {
	fn := func(token string) (identity.Identity, error) {
		tokenInfo, err := grpAuthClient.VerifyToken(context.Background(), &authproto.VerifyTokenRequest{
			Token: token,
		})
		if err != nil {
			return identity.NullIdentity, errors.Wrap(fmt.Errorf("%w: Could not verify token: %s", application.ErrUnauthorized, err))
		}

		user, err := repository.Get(context.Background(), tokenInfo.GetUserId())
		if err != nil {
			return identity.NullIdentity, errors.Wrap(err)
		}

		i := identity.Identity{
			ID:    uuid.MustParse(user.GetID()),
			Token: token,
			Email: user.GetEmail(),
			Roles: []string{"USER"},
		}

		return i, nil
	}

	return fn
}
