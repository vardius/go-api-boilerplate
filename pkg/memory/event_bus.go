package memory

import (
	"context"

	"github.com/vardius/go-api-boilerplate/pkg/domain"

	"github.com/vardius/golog"
	messagebus "github.com/vardius/message-bus"
)

type eventBus struct {
	messageBus messagebus.MessageBus
	logger     golog.Logger
}

func (bus *eventBus) Publish(ctx context.Context, eventType string, event domain.Event) {
	bus.logger.Debug(ctx, "[API EventBus|Publish]: %s %q\n", eventType, event.Payload)
	bus.messageBus.Publish(eventType, ctx, event)
}

func (bus *eventBus) Subscribe(eventType string, fn domain.EventHandler) error {
	bus.logger.Info(nil, "[API EventBus|Subscribe]: %s\n", eventType)
	return bus.messageBus.Subscribe(eventType, fn)
}

func (bus *eventBus) Unsubscribe(eventType string, fn domain.EventHandler) error {
	bus.logger.Info(nil, "[API EventBus|Unsubscribe]: %s\n", eventType)
	return bus.messageBus.Unsubscribe(eventType, fn)
}

// NewEventBus creates in memory event bus
func NewEventBus(log golog.Logger) domain.EventBus {
	return &eventBus{messageBus, log}
}
