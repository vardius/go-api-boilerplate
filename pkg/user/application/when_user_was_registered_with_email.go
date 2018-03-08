package application

import (
	"context"
	"log"

	"github.com/vardius/go-api-boilerplate/pkg/common/domain"
)

// WhenUserWasRegisteredWithEmail handles event
func WhenUserWasRegisteredWithEmail(ctx context.Context, event domain.Event) {
	// todo: register user and send email with auth token
	log.Printf("[user EventHandler] %s", event.Payload)
}
