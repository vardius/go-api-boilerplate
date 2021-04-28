package eventhandler

import (
	"context"
	"time"

	"github.com/dgrijalva/jwt-go"

	"github.com/vardius/go-api-boilerplate/cmd/auth/proto"
	"github.com/vardius/go-api-boilerplate/cmd/user/internal/application/config"
	"github.com/vardius/go-api-boilerplate/cmd/user/internal/application/services/mailer"
	"github.com/vardius/go-api-boilerplate/cmd/user/internal/domain/user"
	"github.com/vardius/go-api-boilerplate/cmd/user/internal/infrastructure/persistence"
	"github.com/vardius/go-api-boilerplate/pkg/auth"
	"github.com/vardius/go-api-boilerplate/pkg/domain"
	apperrors "github.com/vardius/go-api-boilerplate/pkg/errors"
	"github.com/vardius/go-api-boilerplate/pkg/eventbus"
	"github.com/vardius/go-api-boilerplate/pkg/executioncontext"
	"github.com/vardius/go-api-boilerplate/pkg/identity"
)

// WhenUserAccessTokenWasRequested handles event
func WhenUserAccessTokenWasRequested(cfg *config.Config, signedMethod jwt.SigningMethod, authenticator auth.Authenticator, userRepository persistence.UserRepository, authClient proto.AuthenticationServiceClient) eventbus.EventHandler {
	fn := func(parentCtx context.Context, event *domain.Event) error {
		if !executioncontext.Has(parentCtx, executioncontext.LIVE) {
			return nil
		}

		ctx, cancel := context.WithTimeout(parentCtx, time.Second*120)
		defer cancel()

		e := event.Payload.(*user.AccessTokenWasRequested)

		u, err := userRepository.Get(ctx, e.ID.String())
		if err != nil {
			return apperrors.Wrap(err)
		}

		var permissions identity.Permission
		permissions = permissions.Add(identity.PermissionUserRead)
		permissions = permissions.Add(identity.PermissionUserWrite)
		permissions = permissions.Add(identity.PermissionClientRead)
		permissions = permissions.Add(identity.PermissionClientWrite)
		permissions = permissions.Add(identity.PermissionTokenRead)

		i := identity.Identity{
			Permission: permissions,
			UserID:     e.ID,
		}
		claims := &auth.Claims{
			StandardClaims: jwt.StandardClaims{
				Subject:   u.GetID(),
				ExpiresAt: time.Now().Add(365 * 24 * time.Hour).Unix(),
			},
			Identity: &i,
		}

		token := jwt.NewWithClaims(signedMethod, claims)

		accessToken, err := authenticator.Sign(token)
		if err != nil {
			return apperrors.Wrap(err)
		}
		i.Token = accessToken

		const createTokenCommandName = "token-create"
		if _, err := authClient.DispatchTokenCommand(identity.ContextWithIdentity(ctx, &i), &proto.DispatchAuthCommandRequest{
			Name:    createTokenCommandName,
			Payload: nil,
		}); err != nil {
			return apperrors.Wrap(err)
		}

		if err := mailer.SendLoginEmail(ctx, cfg, string(e.Email), accessToken, e.RedirectPath); err != nil {
			return apperrors.Wrap(err)
		}

		return nil
	}

	return fn
}
