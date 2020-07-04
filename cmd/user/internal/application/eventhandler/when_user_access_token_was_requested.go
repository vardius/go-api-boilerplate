package eventhandler

import (
	"context"
	"encoding/json"

	"github.com/vardius/go-api-boilerplate/cmd/user/internal/application/config"
	"github.com/vardius/go-api-boilerplate/cmd/user/internal/application/mailer"
	"github.com/vardius/go-api-boilerplate/cmd/user/internal/domain/user"
	"github.com/vardius/go-api-boilerplate/pkg/auth/oauth2"
	"github.com/vardius/go-api-boilerplate/pkg/domain"
	"github.com/vardius/go-api-boilerplate/pkg/eventbus"
	"github.com/vardius/go-api-boilerplate/pkg/log"
)

// WhenUserAccessTokenWasRequested handles event
func WhenUserAccessTokenWasRequested(tokenProvider oauth2.TokenProvider) eventbus.EventHandler {
	fn := func(ctx context.Context, event domain.Event) {
		logger := log.New(config.Env.App.Environment)
		logger.Info(ctx, "[EventHandler] %s\n", event.Payload)

		e := user.WasRegisteredWithEmail{}

		err := json.Unmarshal(event.Payload, &e)
		if err != nil {
			logger.Error(ctx, "[EventHandler] WhenUserAccessTokenWasRequested: %v\n", err)
			return
		}

		token, err := tokenProvider.RetrieveToken(ctx, string(e.Email))
		if err != nil {
			logger.Error(ctx, "[EventHandler] WhenUserAccessTokenWasRequested: %v\n", err)
			return
		}

		if err := mailer.SendLoginEmail(ctx, string(e.Email), token.AccessToken); err != nil {
			logger.Error(ctx, "[EventHandler] WhenUserAccessTokenWasRequested: %v\n", err)
			return
		}
	}

	return fn
}
