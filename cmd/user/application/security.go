package application

import (
	"context"

	"github.com/google/uuid"
	auth_proto "github.com/vardius/go-api-boilerplate/cmd/auth/infrastructure/proto"
	user_persistance "github.com/vardius/go-api-boilerplate/cmd/user/infrastructure/persistence"
	http_authenticator "github.com/vardius/go-api-boilerplate/pkg/http/security/authenticator"
	"github.com/vardius/go-api-boilerplate/pkg/identity"
	"github.com/vardius/golog"
)

// TokenAuthHandler provides token auth function
func TokenAuthHandler(grpAuthClient auth_proto.AuthenticationServiceClient, repository user_persistance.UserRepository, logger golog.Logger) http_authenticator.TokenAuthFunc {
	fn := func(token string) (*identity.Identity, error) {
		tokenInfo, err := grpAuthClient.VerifyToken(context.Background(), &auth_proto.VerifyTokenRequest{
			Token: token,
		})
		if err != nil {
			logger.Error(context.Background(), "TokenAuthHandler Error: %v\n", err)
			return nil, err
		}

		user, err := repository.Get(context.Background(), tokenInfo.GetUserId())
		if err != nil {
			logger.Error(context.Background(), "TokenAuthHandler Error: %v\n", err)
			return nil, err
		}

		i := &identity.Identity{
			ID:    uuid.MustParse(user.Id),
			Token: token,
			Email: user.Email,
			Roles: []string{"USER"},
		}

		return i, nil
	}

	return http_authenticator.TokenAuthFunc(fn)
}
