package user

import (
	"app/pkg/auth"
	"app/pkg/domain"
	"fmt"
)

// Domain name
const Domain = "users"

func registerCommandHandlers(commandBus domain.CommandBus, repository *eventSourcedRepository, jwtService auth.JwtService) {
	commandBus.Subscribe(Domain+RegisterWithEmail, onRegisterWithEmail(repository, jwtService))
	commandBus.Subscribe(Domain+RegisterWithGoogle, onRegisterWithGoogle(repository))
	commandBus.Subscribe(Domain+RegisterWithFacebook, onRegisterWithFacebook(repository))
	commandBus.Subscribe(Domain+ChangeEmailAddress, onChangeEmailAddress(repository))
}

func registerEventHandlers(eventBus domain.EventBus) {
	eventBus.Subscribe(fmt.Sprintf("%T", &WasRegisteredWithEmail{}), onWasRegisteredWithEmail)
	eventBus.Subscribe(fmt.Sprintf("%T", &WasRegisteredWithGoogle{}), onWasRegisteredWithGoogle)
	eventBus.Subscribe(fmt.Sprintf("%T", &WasRegisteredWithFacebook{}), onWasRegisteredWithFacebook)
	eventBus.Subscribe(fmt.Sprintf("%T", &EmailAddressWasChanged{}), onEmailAddressWasChanged)
}

// Init user domain
func Init(eventStore domain.EventStore, eventBus domain.EventBus, commandBus domain.CommandBus, jwtService auth.JwtService) {
	repository := newRepository(fmt.Sprintf("%T", User{}), eventStore, eventBus)

	registerCommandHandlers(commandBus, repository, jwtService)
	registerEventHandlers(eventBus)
}
