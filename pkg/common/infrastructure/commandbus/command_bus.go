package commandbus

import (
	"context"
	"encoding/json"
)

// CommandHandler function
type CommandHandler func(ctx context.Context, payload json.RawMessage, out chan<- error)

// CommandBus allows to subscribe/dispatch commands
type CommandBus interface {
	Publish(ctx context.Context, command string, payload json.RawMessage, out chan<- error)
	Subscribe(command string, fn CommandHandler) error
	Unsubscribe(command string, fn CommandHandler) error
}
