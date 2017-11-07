package eventbus

import (
	"context"

	"github.com/vardius/go-api-boilerplate/pkg/domain"
	"github.com/vardius/golog"
	messagebus "github.com/vardius/message-bus"
)

type eventBus struct {
	messageBus messagebus.MessageBus
}

func (bus *eventBus) Publish(ctx context.Context, eventType string, event domain.Event) {
	bus.messageBus.Publish(eventType, ctx, event)
}

func (bus *eventBus) Subscribe(eventType string, fn domain.EventHandler) error {
	return bus.messageBus.Subscribe(eventType, fn)
}

func (bus *eventBus) Unsubscribe(eventType string, fn domain.EventHandler) error {
	return bus.messageBus.Unsubscribe(eventType, fn)
}

// New creates in memory event bus
func New() domain.EventBus {
	return &eventBus{messagebus.New()}
}

type loggableEventBus struct {
	eventBus domain.EventBus
	logger   golog.Logger
}

func (bus *loggableEventBus) Publish(ctx context.Context, eventType string, event domain.Event) {
	bus.logger.Debug(ctx, "[API EventBus|Publish]: %s %q\n", eventType, event.Payload)
	bus.eventBus.Publish(ctx, eventType, event)
}

func (bus *loggableEventBus) Subscribe(eventType string, fn domain.EventHandler) error {
	bus.logger.Info(nil, "[API EventBus|Subscribe]: %s\n", eventType)
	return bus.eventBus.Subscribe(eventType, fn)
}

func (bus *loggableEventBus) Unsubscribe(eventType string, fn domain.EventHandler) error {
	bus.logger.Info(nil, "[API EventBus|Unsubscribe]: %s\n", eventType)
	return bus.eventBus.Unsubscribe(eventType, fn)
}

// WithLogger creates in memory event bus
func WithLogger(parent domain.EventBus, log golog.Logger) domain.EventBus {
	return &loggableEventBus{parent, log}
}
