package eventbus

import (
	"context"

	"github.com/vardius/go-api-boilerplate/pkg/domain"
)

// EventHandler function
type EventHandler func(ctx context.Context, event *domain.Event) error

// EventBus interface
// event bus is different to command bus by allowing multiple handlers to the same topic
type EventBus interface {
	Publish(ctx context.Context, event *domain.Event) error
	Subscribe(ctx context.Context, eventType string, fn EventHandler) error
	Unsubscribe(ctx context.Context, eventType string, fn EventHandler) error

	// PublishAndAcknowledge blocks and returns grouped error after all handlers are executed
	PublishAndAcknowledge(parentCtx context.Context, event *domain.Event) error
}
