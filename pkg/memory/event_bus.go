package memory

import (
	"app/pkg/domain"
	"context"

	"github.com/vardius/golog"
	messagebus "github.com/vardius/message-bus"
)

type eventBus struct {
	messageBus messagebus.MessageBus
	logger     golog.Logger
}

func (bus *eventBus) Publish(eventType string, ctx context.Context, event *domain.Event) {
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

func NewEventBus(log golog.Logger) domain.EventBus {
	return &eventBus{messageBus, log}
}
