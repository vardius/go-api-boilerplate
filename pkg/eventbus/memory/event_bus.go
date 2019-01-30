/*
Package eventbus provides memory implementation of domain event store
*/
package eventbus

import (
	"context"

	"github.com/vardius/go-api-boilerplate/pkg/domain"
	baseeventbus "github.com/vardius/go-api-boilerplate/pkg/eventbus"
	"github.com/vardius/golog"
	messagebus "github.com/vardius/message-bus"
)

type eventBus struct {
	messageBus messagebus.MessageBus
}

func (bus *eventBus) Publish(ctx context.Context, eventType string, event domain.Event) {
	bus.messageBus.Publish(eventType, ctx, event)
}

func (bus *eventBus) Subscribe(eventType string, fn baseeventbus.EventHandler) error {
	return bus.messageBus.Subscribe(eventType, fn)
}

func (bus *eventBus) Unsubscribe(eventType string, fn baseeventbus.EventHandler) error {
	return bus.messageBus.Unsubscribe(eventType, fn)
}

// New creates in memory event bus
func New(maxConcurrentCalls int) baseeventbus.EventBus {
	return &eventBus{messagebus.New(maxConcurrentCalls)}
}

type loggableEventBus struct {
	eventBus baseeventbus.EventBus
	logger   golog.Logger
}

func (bus *loggableEventBus) Publish(ctx context.Context, eventType string, event domain.Event) {
	bus.logger.Debug(ctx, "[EventBus|Publish]: %s %s\n", eventType, event.Payload)
	bus.eventBus.Publish(ctx, eventType, event)
}

func (bus *loggableEventBus) Subscribe(eventType string, fn baseeventbus.EventHandler) error {
	bus.logger.Info(nil, "[EventBus|Subscribe]: %s\n", eventType)
	return bus.eventBus.Subscribe(eventType, fn)
}

func (bus *loggableEventBus) Unsubscribe(eventType string, fn baseeventbus.EventHandler) error {
	bus.logger.Info(nil, "[EventBus|Unsubscribe]: %s\n", eventType)
	return bus.eventBus.Unsubscribe(eventType, fn)
}

// WithLogger creates in memory event bus
func WithLogger(parent baseeventbus.EventBus, log golog.Logger) baseeventbus.EventBus {
	return &loggableEventBus{parent, log}
}

// NewLoggable creates in memory event bus with logger
func NewLoggable(maxConcurrentCalls int, log golog.Logger) baseeventbus.EventBus {
	return &loggableEventBus{New(maxConcurrentCalls), log}
}
