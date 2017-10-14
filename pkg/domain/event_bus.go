package domain

import (
	"context"
)

type EventHandler func(ctx context.Context, event *Event)

type EventBus interface {
	Publish(eventType string, ctx context.Context, event *Event)
	Subscribe(eventType string, fn EventHandler) error
	Unsubscribe(eventType string, fn EventHandler) error
}
