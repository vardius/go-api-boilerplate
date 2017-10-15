package user

import (
	"app/pkg/domain"
	"context"
	"reflect"
)

func registerEventHandlers(eventBus domain.EventBus) {
	eventBus.Subscribe(WasRegisteredWithEmailType, handleUserWasRegisteredWithEmail)
	eventBus.Subscribe(WasRegisteredWithGoogleType, handleUserWasRegisteredWithGoogle)
	eventBus.Subscribe(WasRegisteredWithFacebookType, handleUserWasRegisteredWithFacebook)
	eventBus.Subscribe(EmailAddressWasChangedType, handleUserEmailAddressWasChanged)
}

func handleUserWasRegisteredWithEmail(ctx context.Context, event *domain.Event) {
	// todo: register user and send email with auth token
	logger.Error(nil, "handle UserWasRegisteredWithEmail %v\n", event)
}

func handleUserWasRegisteredWithGoogle(ctx context.Context, event *domain.Event) {
	// todo: register user
	logger.Error(nil, "handle UserWasRegisteredWithGoogle %v\n", event)
}

func handleUserWasRegisteredWithFacebook(ctx context.Context, event *domain.Event) {
	// todo: register user
	logger.Error(nil, "handle UserWasRegisteredWithFacebook %v\n", event)
}

func handleUserEmailAddressWasChanged(ctx context.Context, event *domain.Event) {
	// todo: register user
	logger.Error(nil, "handle UserEmailAddressWasChanged %v\n", event)
}
