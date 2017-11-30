package user

import (
	"fmt"

	"github.com/vardius/go-api-boilerplate/pkg/auth/jwt"
	"github.com/vardius/go-api-boilerplate/pkg/domain"
)

const commandPrefix = "users"

func registerCommandHandlers(commandBus domain.CommandBus, repository *eventSourcedRepository, j jwt.Jwt) {
	commandBus.Subscribe(commandPrefix+RegisterWithEmail, onRegisterWithEmail(repository, j))
	commandBus.Subscribe(commandPrefix+RegisterWithGoogle, onRegisterWithGoogle(repository))
	commandBus.Subscribe(commandPrefix+RegisterWithFacebook, onRegisterWithFacebook(repository))
	commandBus.Subscribe(commandPrefix+ChangeEmailAddress, onChangeEmailAddress(repository))
}

func registerEventHandlers(eventBus domain.EventBus) {
	eventBus.Subscribe(fmt.Sprintf("%T", &WasRegisteredWithEmail{}), onWasRegisteredWithEmail)
	eventBus.Subscribe(fmt.Sprintf("%T", &WasRegisteredWithGoogle{}), onWasRegisteredWithGoogle)
	eventBus.Subscribe(fmt.Sprintf("%T", &WasRegisteredWithFacebook{}), onWasRegisteredWithFacebook)
	eventBus.Subscribe(fmt.Sprintf("%T", &EmailAddressWasChanged{}), onEmailAddressWasChanged)
}

// Init user domain
func Init(eventStore domain.EventStore, eventBus domain.EventBus, commandBus domain.CommandBus, j jwt.Jwt) {
	repository := newRepository(fmt.Sprintf("%T", User{}), eventStore, eventBus)

	registerCommandHandlers(commandBus, repository, j)
	registerEventHandlers(eventBus)
}
