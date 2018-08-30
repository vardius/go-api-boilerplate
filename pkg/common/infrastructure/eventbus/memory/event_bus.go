/*
Package eventbus provides memory implementation of domain event store
*/
package eventbus

import (
	"context"

	"github.com/vardius/go-api-boilerplate/pkg/common/domain"
	baseeventbus "github.com/vardius/go-api-boilerplate/pkg/common/infrastructure/eventbus"
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
	serverName string
	eventBus   baseeventbus.EventBus
	logger     golog.Logger
}

func (bus *loggableEventBus) Publish(ctx context.Context, eventType string, event domain.Event) {
	bus.logger.Debug(ctx, "[%s EventBus|Publish]: %s %+v\n", bus.serverName, eventType, event.Payload)
	bus.eventBus.Publish(ctx, eventType, event)
}

func (bus *loggableEventBus) Subscribe(eventType string, fn baseeventbus.EventHandler) error {
	bus.logger.Info(nil, "[%s EventBus|Subscribe]: %s\n", bus.serverName, eventType)
	return bus.eventBus.Subscribe(eventType, fn)
}

func (bus *loggableEventBus) Unsubscribe(eventType string, fn baseeventbus.EventHandler) error {
	bus.logger.Info(nil, "[%s EventBus|Unsubscribe]: %s\n", bus.serverName, eventType)
	return bus.eventBus.Unsubscribe(eventType, fn)
}

// WithLogger creates in memory event bus
func WithLogger(serverName string, parent baseeventbus.EventBus, log golog.Logger) baseeventbus.EventBus {
	return &loggableEventBus{serverName, parent, log}
}

// NewLoggable creates in memory event bus with logger
func NewLoggable(maxConcurrentCalls int, serverName string, log golog.Logger) baseeventbus.EventBus {
	return &loggableEventBus{serverName, New(maxConcurrentCalls), log}
}
