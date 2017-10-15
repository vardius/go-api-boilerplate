package user

import (
	"app/pkg/auth"
	"app/pkg/domain"
	"fmt"

	"github.com/vardius/golog"
)

var logger golog.Logger

const Domain = "users"

// Init setup domain
func Init(eventStore domain.EventStore, eventBus domain.EventBus, commandBus domain.CommandBus, jwtService auth.JwtService, log golog.Logger) {
	logger = log
	streamName := fmt.Sprintf("%T", User{})
	eventSourcedRepository := newEventSourcedRepository(streamName, eventStore, eventBus)

	registerCommandHandlers(commandBus, eventSourcedRepository, jwtService)
	registerEventHandlers(eventBus)
}

func registerCommandHandlers(commandBus domain.CommandBus, repository *eventSourcedRepository, jwtService auth.JwtService) {
	commandBus.Subscribe(Domain+"-"+RegisterWithEmail, registerUserWithEmail(repository, jwtService))
	commandBus.Subscribe(Domain+"-"+RegisterWithGoogle, registerUserWithGoogle(repository))
	commandBus.Subscribe(Domain+"-"+RegisterWithFacebook, registerUserWithFacebook(repository))
	commandBus.Subscribe(Domain+"-"+ChangeEmailAddress, changeUserEmailAddress(repository))
}

func registerEventHandlers(eventBus domain.EventBus) {
	eventBus.Subscribe(WasRegisteredWithEmailType, handleUserWasRegisteredWithEmail)
	eventBus.Subscribe(WasRegisteredWithGoogleType, handleUserWasRegisteredWithGoogle)
	eventBus.Subscribe(WasRegisteredWithFacebookType, handleUserWasRegisteredWithFacebook)
	eventBus.Subscribe(EmailAddressWasChangedType, handleUserEmailAddressWasChanged)
}
