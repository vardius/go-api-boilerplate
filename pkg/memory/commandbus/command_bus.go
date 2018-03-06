/*
Package commandbus provides memory implementation of domain event store
*/
package commandbus

import (
	"context"
	"encoding/json"

	"github.com/vardius/go-api-boilerplate/pkg/domain"
	"github.com/vardius/golog"
	messagebus "github.com/vardius/message-bus"
)

type commandBus struct {
	messageBus messagebus.MessageBus
}

func (bus *commandBus) Publish(ctx context.Context, command string, payload json.RawMessage, out chan<- error) {
	bus.messageBus.Publish(command, ctx, payload, out)
}

func (bus *commandBus) Subscribe(command string, fn domain.CommandHandler) error {
	return bus.messageBus.Subscribe(command, fn)
}

func (bus *commandBus) Unsubscribe(command string, fn domain.CommandHandler) error {
	return bus.messageBus.Unsubscribe(command, fn)
}

// New creates in memory command bus
func New() domain.CommandBus {
	return &commandBus{messagebus.New()}
}

type loggableCommandBus struct {
	serverName string
	commandBus domain.CommandBus
	logger     golog.Logger
}

func (bus *loggableCommandBus) Publish(ctx context.Context, command string, payload json.RawMessage, out chan<- error) {
	bus.logger.Debug(ctx, "[%s CommandBus|Publish]: %s %s\n", bus.serverName, command, payload)
	bus.commandBus.Publish(ctx, command, payload, out)
}

func (bus *loggableCommandBus) Subscribe(command string, fn domain.CommandHandler) error {
	bus.logger.Info(nil, "[%s CommandBus|Subscribe]: %s\n", bus.serverName, command)
	return bus.commandBus.Subscribe(command, fn)
}

func (bus *loggableCommandBus) Unsubscribe(command string, fn domain.CommandHandler) error {
	bus.logger.Info(nil, "[%s CommandBus|Unsubscribe]: %s\n", bus.serverName, command)
	return bus.commandBus.Unsubscribe(command, fn)
}

// WithLogger creates loggable inmemory command bus
func WithLogger(serverName string, parent domain.CommandBus, log golog.Logger) domain.CommandBus {
	return &loggableCommandBus{serverName, parent, log}
}
