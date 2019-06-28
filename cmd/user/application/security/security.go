package application

import (
	"context"

	"github.com/google/uuid"
	auth_proto "github.com/vardius/go-api-boilerplate/cmd/auth/infrastructure/proto"
	user_persistance "github.com/vardius/go-api-boilerplate/cmd/user/infrastructure/persistence"
	"github.com/vardius/go-api-boilerplate/pkg/errors"
	http_authenticator "github.com/vardius/go-api-boilerplate/pkg/http/security/authenticator"
	"github.com/vardius/go-api-boilerplate/pkg/identity"
)

// TokenAuthHandler provides token auth function
func TokenAuthHandler(grpAuthClient auth_proto.AuthenticationServiceClient, repository user_persistance.UserRepository) http_authenticator.TokenAuthFunc {
	fn := func(token string) (identity.Identity, error) {
		tokenInfo, err := grpAuthClient.VerifyToken(context.Background(), &auth_proto.VerifyTokenRequest{
			Token: token,
		})
		if err != nil {
			return identity.NullIdentity, errors.Wrap(err, errors.UNAUTHORIZED, "Could not verify token")
		}

		user, err := repository.Get(context.Background(), tokenInfo.GetUserId())
		if err != nil {
			return identity.NullIdentity, errors.Wrap(err, errors.INTERNAL, "Could not find user for token")
		}

		i := identity.Identity{
			ID:    uuid.MustParse(user.GetID()),
			Token: token,
			Email: user.GetEmail(),
			Roles: []string{"USER"},
		}

		return i, nil
	}

	return http_authenticator.TokenAuthFunc(fn)
}
