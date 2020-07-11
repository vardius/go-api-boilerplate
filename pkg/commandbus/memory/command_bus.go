package memory

import (
	"context"

	messagebus "github.com/vardius/message-bus"

	"github.com/vardius/go-api-boilerplate/pkg/commandbus"
	"github.com/vardius/go-api-boilerplate/pkg/domain"
	"github.com/vardius/go-api-boilerplate/pkg/log"
)

// New creates in memory command bus
func New(maxConcurrentCalls int, logger *log.Logger) commandbus.CommandBus {
	return &commandBus{messagebus.New(maxConcurrentCalls), logger}
}

type commandBus struct {
	messageBus messagebus.MessageBus
	logger     *log.Logger
}

func (bus *commandBus) Publish(ctx context.Context, command domain.Command, out chan<- error) {
	bus.logger.Debug(ctx, "[CommandBus] Publish: %s %+v\n", command.GetName(), command)
	bus.messageBus.Publish(command.GetName(), ctx, command, out)
}

func (bus *commandBus) Subscribe(ctx context.Context, commandName string, fn commandbus.CommandHandler) error {
	bus.logger.Info(nil, "[CommandBus] Subscribe: %s\n", commandName)
	return bus.messageBus.Subscribe(commandName, fn)
}

func (bus *commandBus) Unsubscribe(ctx context.Context, commandName string, fn commandbus.CommandHandler) error {
	bus.logger.Info(nil, "[CommandBus] Unsubscribe: %s\n", commandName)
	return bus.messageBus.Unsubscribe(commandName, fn)
}
