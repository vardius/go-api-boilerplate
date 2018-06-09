package eventbus

import (
	"context"

	"github.com/vardius/go-api-boilerplate/pkg/common/domain"
)

// EventHandler function
type EventHandler func(ctx context.Context, event domain.Event)

// EventBus allow to publis/subscribe to events
type EventBus interface {
	Publish(ctx context.Context, eventType string, event domain.Event)
	Subscribe(eventType string, fn EventHandler) error
	Unsubscribe(eventType string, fn EventHandler) error
}
