package memory

import (
	"context"
	"time"

	messagebus "github.com/vardius/message-bus"

	"github.com/vardius/go-api-boilerplate/pkg/domain"
	"github.com/vardius/go-api-boilerplate/pkg/eventbus"
	"github.com/vardius/go-api-boilerplate/pkg/log"
)

// New creates memory event bus
func New(handlerTimeout time.Duration, maxConcurrentCalls int, log *log.Logger) eventbus.EventBus {
	return &eventBus{handlerTimeout, messagebus.New(maxConcurrentCalls), log}
}

type eventBus struct {
	handlerTimeout time.Duration
	messageBus     messagebus.MessageBus
	logger         *log.Logger
}

func (bus *eventBus) Publish(parentCtx context.Context, event domain.Event) error {
	ctx, cancel := context.WithTimeout(context.Background(), bus.handlerTimeout)
	defer cancel()

	bus.logger.Debug(parentCtx, "[EventBus] Publish: %s %+v\n", event.Metadata.Type, event)
	bus.messageBus.Publish(event.Metadata.Type, ctx, event)

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
