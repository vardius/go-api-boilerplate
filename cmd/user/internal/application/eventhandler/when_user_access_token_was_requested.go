package eventhandler

import (
	"context"
	"encoding/json"
	"time"

	"github.com/vardius/go-api-boilerplate/cmd/user/internal/application/config"
	"github.com/vardius/go-api-boilerplate/cmd/user/internal/application/mailer"
	"github.com/vardius/go-api-boilerplate/cmd/user/internal/domain/user"
	"github.com/vardius/go-api-boilerplate/pkg/auth/oauth2"
	"github.com/vardius/go-api-boilerplate/pkg/domain"
	"github.com/vardius/go-api-boilerplate/pkg/errors"
	"github.com/vardius/go-api-boilerplate/pkg/eventbus"
	"github.com/vardius/go-api-boilerplate/pkg/log"
)

// WhenUserAccessTokenWasRequested handles event
func WhenUserAccessTokenWasRequested(tokenProvider oauth2.TokenProvider) eventbus.EventHandler {
	fn := func(parentCtx context.Context, event domain.Event) {
		ctx, cancel := context.WithTimeout(parentCtx, time.Second*120)
		defer cancel()

		logger := log.New(config.Env.App.Environment)
		logger.Info(ctx, "[EventHandler] %s\n", event.Payload)

		e := user.WasRegisteredWithEmail{}
		if err := json.Unmarshal(event.Payload, &e); err != nil {
			logger.Error(ctx, "[EventHandler] WhenUserAccessTokenWasRequested: %v\n", errors.Wrap(err))
			return
		}

		token, err := tokenProvider.RetrieveToken(ctx, string(e.Email))
		if err != nil {
			logger.Error(ctx, "[EventHandler] WhenUserAccessTokenWasRequested: %v\n", errors.Wrap(err))
			return
		}

		if err := mailer.SendLoginEmail(ctx, "WhenUserAccessTokenWasRequested", string(e.Email), token.AccessToken); err != nil {
			logger.Error(ctx, "[EventHandler] WhenUserAccessTokenWasRequested: %v\n", errors.Wrap(err))
			return
		}
	}

	return fn
}
