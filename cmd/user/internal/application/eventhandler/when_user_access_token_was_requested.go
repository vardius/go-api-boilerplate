package eventhandler

import (
	"context"
	"encoding/json"
	"time"

	appidentity "github.com/vardius/go-api-boilerplate/cmd/user/internal/application/identity"
	"github.com/vardius/go-api-boilerplate/cmd/user/internal/application/mailer"
	"github.com/vardius/go-api-boilerplate/cmd/user/internal/domain/user"
	"github.com/vardius/go-api-boilerplate/pkg/auth/oauth2"
	"github.com/vardius/go-api-boilerplate/pkg/domain"
	apperrors "github.com/vardius/go-api-boilerplate/pkg/errors"
	"github.com/vardius/go-api-boilerplate/pkg/eventbus"
	"github.com/vardius/go-api-boilerplate/pkg/executioncontext"
)

// WhenUserAccessTokenWasRequested handles event
func WhenUserAccessTokenWasRequested(tokenProvider oauth2.TokenProvider, identityProvider appidentity.Provider) eventbus.EventHandler {
	fn := func(parentCtx context.Context, event domain.Event) error {
		ctx, cancel := context.WithTimeout(parentCtx, time.Second*120)
		defer cancel()

		e := user.WasRegisteredWithEmail{}
		if err := json.Unmarshal(event.Payload, &e); err != nil {
			return apperrors.Wrap(err)
		}

		i, err := identityProvider.GetByUserEmail(ctx, e.Email.String())
		if err != nil {
			return apperrors.Wrap(err)
		}

		token, err := tokenProvider.RetrievePasswordCredentialsToken(ctx, i.ClientID.String(), i.ClientSecret.String(), string(e.Email), oauth2.AllScopes)
		if err != nil {
			return apperrors.Wrap(err)
		}

		if executioncontext.Has(ctx, executioncontext.LIVE) {
			if err := mailer.SendLoginEmail(ctx, "WhenUserAccessTokenWasRequested", string(e.Email), token.AccessToken, e.RedirectPath); err != nil {
				return apperrors.Wrap(err)
			}
		}

		return nil
	}

	return fn
}
