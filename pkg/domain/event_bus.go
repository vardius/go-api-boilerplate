package domain

import (
	"context"
)

// EventHandler function
type EventHandler func(ctx context.Context, event Event)

// EventBus allow to publis/subscribe to events
type EventBus interface {
	Publish(eventType string, ctx context.Context, event Event)
	Subscribe(eventType string, fn EventHandler) error
	Unsubscribe(eventType string, fn EventHandler) error
}
