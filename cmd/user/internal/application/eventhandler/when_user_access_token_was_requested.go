package eventhandler

import (
	"context"
	"encoding/json"
	"log"

	"golang.org/x/oauth2"

	"github.com/vardius/go-api-boilerplate/cmd/user/internal/domain/user"
	"github.com/vardius/go-api-boilerplate/internal/domain"
	"github.com/vardius/go-api-boilerplate/internal/eventbus"
)

// WhenUserAccessTokenWasRequested handles event
func WhenUserAccessTokenWasRequested(config oauth2.Config, secretKey string) eventbus.EventHandler {
	fn := func(ctx context.Context, event domain.Event) {
		// this goroutine runs independently to request's goroutine,
		// there for recover middlewears will not recover from panic to prevent crash
		defer recoverEventHandler()

		log.Printf("[EventHandler] %s\n", event.Payload)

		e := user.WasRegisteredWithEmail{}

		err := json.Unmarshal(event.Payload, &e)
		if err != nil {
			log.Printf("[EventHandler] Error: %v\n", err)
			return
		}

		token, err := config.PasswordCredentialsToken(ctx, e.Email, e.Password)
		if err != nil {
			log.Printf("[EventHandler] Error: %v\n", err)
			return
		}

		b, err := json.Marshal(token)
		if err != nil {
			log.Printf("[EventHandler] Error: %v\n", err)
			return
		}

		// @TODO: send token with an email as magic link
		log.Printf("[EventHandler] Access Token: %s\n", string(b))
	}

	return fn
}
