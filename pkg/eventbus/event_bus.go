package eventbus

import (
	"context"

	"github.com/vardius/go-api-boilerplate/pkg/domain"
)

// EventHandler function
type EventHandler func(ctx context.Context, event domain.Event)

// EventBus intrface
type EventBus interface {
	Publish(ctx context.Context, event domain.Event)
	Subscribe(ctx context.Context, eventType string, fn EventHandler) error
	Unsubscribe(ctx context.Context, eventType string, fn EventHandler) error
}
