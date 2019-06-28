package commandbus

import (
	"context"

	"github.com/vardius/go-api-boilerplate/pkg/domain"
	"github.com/vardius/golog"
	messagebus "github.com/vardius/message-bus"
)

// CommandHandler function
type CommandHandler interface{}

// CommandBus allows to subscribe/dispatch commands
type CommandBus interface {
	Publish(ctx context.Context, command domain.Command, out chan<- error)
	Subscribe(commandName string, fn CommandHandler) error
	Unsubscribe(commandName string, fn CommandHandler) error
}

// New creates in memory command bus
func New(maxConcurrentCalls int, logger golog.Logger) CommandBus {
	return &commandBus{messagebus.New(maxConcurrentCalls), logger}
}

type commandBus struct {
	messageBus messagebus.MessageBus
	logger     golog.Logger
}

func (bus *commandBus) Publish(ctx context.Context, command domain.Command, out chan<- error) {
	bus.logger.Debug(ctx, "[CommandBus|Publish]: %s %+v\n", command.GetName(), command)
	bus.messageBus.Publish(command.GetName(), ctx, command, out)
}

func (bus *commandBus) Subscribe(commandName string, fn CommandHandler) error {
	bus.logger.Info(nil, "[CommandBus|Subscribe]: %s\n", commandName)
	return bus.messageBus.Subscribe(commandName, fn)
}

func (bus *commandBus) Unsubscribe(commandName string, fn CommandHandler) error {
	bus.logger.Info(nil, "[CommandBus|Unsubscribe]: %s\n", commandName)
	return bus.messageBus.Unsubscribe(commandName, fn)
}
