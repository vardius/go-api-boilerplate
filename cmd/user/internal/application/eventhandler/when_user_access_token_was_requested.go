package eventhandler

import (
	"context"
	"encoding/json"
	"time"

	"github.com/vardius/go-api-boilerplate/cmd/user/internal/application/config"
	appidentity "github.com/vardius/go-api-boilerplate/cmd/user/internal/application/identity"
	"github.com/vardius/go-api-boilerplate/cmd/user/internal/application/mailer"
	"github.com/vardius/go-api-boilerplate/cmd/user/internal/domain/user"
	"github.com/vardius/go-api-boilerplate/pkg/auth/oauth2"
	"github.com/vardius/go-api-boilerplate/pkg/domain"
	"github.com/vardius/go-api-boilerplate/pkg/errors"
	"github.com/vardius/go-api-boilerplate/pkg/eventbus"
)

// WhenUserAccessTokenWasRequested handles event
func WhenUserAccessTokenWasRequested(tokenProvider oauth2.TokenProvider, identityProvider appidentity.Provider) eventbus.EventHandler {
	fn := func(parentCtx context.Context, event domain.Event) error {
		ctx, cancel := context.WithTimeout(parentCtx, time.Second*120)
		defer cancel()

		e := user.WasRegisteredWithEmail{}
		if err := json.Unmarshal(event.Payload, &e); err != nil {
			return errors.Wrap(err)
		}

		i, err := identityProvider.GetByUserEmail(ctx, e.Email.String(), config.Env.App.Domain)
		if err != nil {
			return errors.Wrap(err)
		}

		token, err := tokenProvider.RetrievePasswordCredentialsToken(ctx, i.ClientID.String(), i.ClientSecret, string(e.Email), []string{"all"})
		if err != nil {
			return errors.Wrap(err)
		}

		if err := mailer.SendLoginEmail(ctx, "WhenUserAccessTokenWasRequested", string(e.Email), token.AccessToken); err != nil {
			return errors.Wrap(err)
		}

		return nil
	}

	return fn
}
