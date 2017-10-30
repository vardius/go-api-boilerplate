package memory

import (
	"context"
	"encoding/json"

	"github.com/vardius/go-api-boilerplate/pkg/domain"

	"github.com/vardius/golog"
	messagebus "github.com/vardius/message-bus"
)

type commandBus struct {
	messageBus messagebus.MessageBus
	logger     golog.Logger
}

func (bus *commandBus) Publish(ctx context.Context, command string, payload json.RawMessage, out chan<- error) {
	bus.logger.Debug(ctx, "[API CommandBus|Publish]: %s %q\n", command, payload)
	bus.messageBus.Publish(command, ctx, payload, out)
}

func (bus *commandBus) Subscribe(command string, fn domain.CommandHandler) error {
	bus.logger.Info(nil, "[API CommandBus|Subscribe]: %s\n", command)
	return bus.messageBus.Subscribe(command, fn)
}

func (bus *commandBus) Unsubscribe(command string, fn domain.CommandHandler) error {
	bus.logger.Info(nil, "[API CommandBus|Unsubscribe]: %s\n", command)
	return bus.messageBus.Unsubscribe(command, fn)
}

// NewCommandBus creates in memory command bus
func NewCommandBus(log golog.Logger) domain.CommandBus {
	return &commandBus{messageBus, log}
}
