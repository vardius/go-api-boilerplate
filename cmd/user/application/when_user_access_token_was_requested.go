package application

import (
	"context"
	"encoding/json"
	"log"

	"github.com/vardius/go-api-boilerplate/cmd/user/domain/user"
	"github.com/vardius/go-api-boilerplate/pkg/domain"
	"github.com/vardius/go-api-boilerplate/pkg/eventbus"
	"golang.org/x/oauth2"
)

// WhenUserAccessTokenWasRequested handles event
func WhenUserAccessTokenWasRequested(config oauth2.Config, secretKey string) eventbus.EventHandler {
	fn := func(ctx context.Context, event domain.Event) {
		// this goroutine runs independently to request's goroutine,
		// there for recover middlewears will not recover from panic to prevent crash
		defer func() {
			if r := recover(); r != nil {
				log.Printf("[EventHandler] Recovered in %v", r)
			}
		}()

		log.Printf("[EventHandler] %s", event.Payload)

		e := &user.WasRegisteredWithGoogle{}

		err := json.Unmarshal(event.Payload, e)
		if err != nil {
			log.Printf("[EventHandler] Error: %v", err)
			return
		}

		token, err := config.PasswordCredentialsToken(ctx, e.Email, secretKey)
		if err != nil {
			log.Printf("[EventHandler] Error: %v", err)
			return
		}

		b, err := json.Marshal(token)
		if err != nil {
			log.Printf("[EventHandler] Error: %v", err)
			return
		}

		// @TODO: send token with an email as magic link
		log.Printf("[EventHandler] Access Token: %s", string(b))
	}

	return eventbus.EventHandler(fn)
}
