package application

import (
	"context"
	"log"

	"github.com/vardius/go-api-boilerplate/pkg/common/domain"
)

// WhenUserEmailAddressWasChanged handles event
func WhenUserEmailAddressWasChanged(ctx context.Context, event domain.Event) {
	// todo: register user
	log.Printf("[user EventHandler] %s", event.Payload)
}
