/*
Package commandbus provides memory implementation of domain event store
*/
package commandbus

import (
	"context"

	basecommandbus "github.com/vardius/go-api-boilerplate/pkg/common/infrastructure/commandbus"
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
	serverName string
	commandBus basecommandbus.CommandBus
	logger     golog.Logger
}

func (bus *loggableCommandBus) Publish(ctx context.Context, commandName string, command interface{}, out chan<- error) {
	bus.logger.Debug(ctx, "[%s CommandBus|Publish]: %s %+v\n", bus.serverName, commandName, command)
	bus.commandBus.Publish(ctx, commandName, command, out)
}

func (bus *loggableCommandBus) Subscribe(commandName string, fn basecommandbus.CommandHandler) error {
	bus.logger.Info(nil, "[%s CommandBus|Subscribe]: %s\n", bus.serverName, commandName)
	return bus.commandBus.Subscribe(commandName, fn)
}

func (bus *loggableCommandBus) Unsubscribe(commandName string, fn basecommandbus.CommandHandler) error {
	bus.logger.Info(nil, "[%s CommandBus|Unsubscribe]: %s\n", bus.serverName, commandName)
	return bus.commandBus.Unsubscribe(commandName, fn)
}

// WithLogger creates loggable in memory command bus
func WithLogger(serverName string, parent basecommandbus.CommandBus, log golog.Logger) basecommandbus.CommandBus {
	return &loggableCommandBus{serverName, parent, log}
}

// NewLoggable creates in memory command bus with logger
func NewLoggable(maxConcurrentCalls int, serverName string, log golog.Logger) basecommandbus.CommandBus {
	return &loggableCommandBus{serverName, New(maxConcurrentCalls), log}
}
