package eventbus

import (
	"context"

	"github.com/vardius/go-api-boilerplate/pkg/domain"
)

// EventHandler function
type EventHandler func(ctx context.Context, event domain.Event)

// EventBus interface
type EventBus interface {
	Publish(ctx context.Context, event domain.Event) error
	Subscribe(ctx context.Context, eventType string, fn EventHandler) error
	Unsubscribe(ctx context.Context, eventType string, fn EventHandler) error
}
