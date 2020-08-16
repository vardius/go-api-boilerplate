package memory

import (
	"context"

	messagebus "github.com/vardius/message-bus"

	"github.com/vardius/go-api-boilerplate/pkg/domain"
	"github.com/vardius/go-api-boilerplate/pkg/eventbus"
	"github.com/vardius/go-api-boilerplate/pkg/log"
)

// New creates memory event bus
func New(maxConcurrentCalls int, log *log.Logger) eventbus.EventBus {
	return &eventBus{messagebus.New(maxConcurrentCalls), log}
}

type eventBus struct {
	messageBus messagebus.MessageBus
	logger     *log.Logger
}

func (bus *eventBus) Publish(parentCtx context.Context, event domain.Event) error {
	bus.logger.Debug(parentCtx, "[EventBus] Publish: %s %+v\n", event.Metadata.Type, event)
	bus.messageBus.Publish(event.Metadata.Type, context.Background(), event)

	return nil
}

func (bus *eventBus) Subscribe(ctx context.Context, eventType string, fn eventbus.EventHandler) error {
	bus.logger.Info(ctx, "[EventBus] Subscribe: %s\n", eventType)
	return bus.messageBus.Subscribe(eventType, fn)
}

func (bus *eventBus) Unsubscribe(ctx context.Context, eventType string, fn eventbus.EventHandler) error {
	bus.logger.Info(ctx, "[EventBus] Unsubscribe: %s\n", eventType)
	return bus.messageBus.Unsubscribe(eventType, fn)
}
