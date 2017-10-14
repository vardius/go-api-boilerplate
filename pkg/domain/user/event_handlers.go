package user

import (
	"app/pkg/domain"
	"context"
	"reflect"
)

var UserWasRegisteredWithEmailType string = reflect.TypeOf((*UserWasRegisteredWithEmail)(nil)).String()
var UserWasRegisteredWithGoogleType string = reflect.TypeOf((*UserWasRegisteredWithGoogle)(nil)).String()
var UserWasRegisteredWithFacebookType string = reflect.TypeOf((*UserWasRegisteredWithFacebook)(nil)).String()
var UserEmailAddressWasChangedType string = reflect.TypeOf((*UserEmailAddressWasChanged)(nil)).String()

func registerEventHandlers(eventBus domain.EventBus) {
	eventBus.Subscribe(UserWasRegisteredWithEmailType, handleUserWasRegisteredWithEmail)
	eventBus.Subscribe(UserWasRegisteredWithGoogleType, handleUserWasRegisteredWithGoogle)
	eventBus.Subscribe(UserWasRegisteredWithFacebookType, handleUserWasRegisteredWithFacebook)
	eventBus.Subscribe(UserEmailAddressWasChangedType, handleUserEmailAddressWasChanged)
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
