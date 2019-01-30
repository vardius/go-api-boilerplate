package commandbus

import (
	"context"
)

// CommandHandler function
type CommandHandler interface{}

// CommandBus allows to subscribe/dispatch commands
type CommandBus interface {
	Publish(ctx context.Context, commandName string, command interface{}, out chan<- error)
	Subscribe(commandName string, fn CommandHandler) error
	Unsubscribe(commandName string, fn CommandHandler) error
}
