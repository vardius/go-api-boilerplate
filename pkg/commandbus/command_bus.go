package commandbus

import (
	"context"

	"github.com/vardius/go-api-boilerplate/pkg/domain"
)

// CommandHandler function
type CommandHandler func(ctx context.Context, command domain.Command) error

// CommandBus allows to subscribe/dispatch commands
// Subscribing to the same command twice will unsubscribe previous handler
// command handler should be one to one
type CommandBus interface {
	Publish(ctx context.Context, command domain.Command) error
	Subscribe(ctx context.Context, commandName string, fn CommandHandler) error
	Unsubscribe(ctx context.Context, commandName string) error
}
