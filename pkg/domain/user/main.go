package user

import (
	"app/pkg/auth"
	"app/pkg/domain"
	"fmt"

	"github.com/vardius/golog"
)

var logger golog.Logger

type domainEvent interface {
	Apply(*User)
}

func Init(eventStore domain.EventStore, eventBus domain.EventBus, commandBus domain.CommandBus, jwtService auth.JwtService, log golog.Logger) {
	logger = log
	streamName := fmt.Sprintf("%T", User{})
	eventSourcedRepository := newEventSourcedRepository(streamName, eventStore, eventBus)

	registerCommandHandlers(commandBus, eventSourcedRepository, jwtService)
	registerEventHandlers(eventBus)
}
