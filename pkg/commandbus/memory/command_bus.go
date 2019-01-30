/*
Package commandbus provides memory implementation of domain event store
*/
package commandbus

import (
	"context"

	basecommandbus "github.com/vardius/go-api-boilerplate/pkg/commandbus"
	"github.com/vardius/golog"
	messagebus "github.com/vardius/message-bus"
)

type commandBus struct {
	messageBus messagebus.MessageBus
}

func (bus *commandBus) Publish(ctx context.Context, commandName string, command interface{}, out chan<- error) {
	bus.messageBus.Publish(commandName, ctx, command, out)
}

func (bus *commandBus) Subscribe(commandName string, fn basecommandbus.CommandHandler) error {
	return bus.messageBus.Subscribe(commandName, fn)
}

func (bus *commandBus) Unsubscribe(commandName string, fn basecommandbus.CommandHandler) error {
	return bus.messageBus.Unsubscribe(commandName, fn)
}

// New creates in memory command bus
func New(maxConcurrentCalls int) basecommandbus.CommandBus {
	return &commandBus{messagebus.New(maxConcurrentCalls)}
}

type loggableCommandBus struct {
	commandBus basecommandbus.CommandBus
	logger     golog.Logger
}

func (bus *loggableCommandBus) Publish(ctx context.Context, commandName string, command interface{}, out chan<- error) {
	bus.logger.Debug(ctx, "[CommandBus|Publish]: %s %+v\n", commandName, command)
	bus.commandBus.Publish(ctx, commandName, command, out)
}

func (bus *loggableCommandBus) Subscribe(commandName string, fn basecommandbus.CommandHandler) error {
	bus.logger.Info(nil, "[CommandBus|Subscribe]: %s\n", commandName)
	return bus.commandBus.Subscribe(commandName, fn)
}

func (bus *loggableCommandBus) Unsubscribe(commandName string, fn basecommandbus.CommandHandler) error {
	bus.logger.Info(nil, "[CommandBus|Unsubscribe]: %s\n", commandName)
	return bus.commandBus.Unsubscribe(commandName, fn)
}

// WithLogger creates loggable in memory command bus
func WithLogger(parent basecommandbus.CommandBus, log golog.Logger) basecommandbus.CommandBus {
	return &loggableCommandBus{parent, log}
}

// NewLoggable creates in memory command bus with logger
func NewLoggable(maxConcurrentCalls int, log golog.Logger) basecommandbus.CommandBus {
	return &loggableCommandBus{New(maxConcurrentCalls), log}
}
