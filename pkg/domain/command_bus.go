package domain

import (
	"context"
	"encoding/json"
)

// CommandHandler function
type CommandHandler func(ctx context.Context, payload json.RawMessage, out chan<- error)

// CommandBus allows to subscribe/dispatch commands
type CommandBus interface {
	Publish(command string, ctx context.Context, payload json.RawMessage, out chan<- error)
	Subscribe(command string, fn CommandHandler) error
	Unsubscribe(command string, fn CommandHandler) error
}
