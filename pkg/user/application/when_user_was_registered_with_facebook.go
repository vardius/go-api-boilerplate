package application

import (
	"context"
	"log"

	"github.com/vardius/go-api-boilerplate/pkg/common/domain"
)

// WhenUserWasRegisteredWithFacebook handles event
func WhenUserWasRegisteredWithFacebook(ctx context.Context, event domain.Event) {
	// todo: register user
	log.Printf("[user EventHandler] %s", event.Payload)
}
