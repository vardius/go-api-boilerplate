package memory

import (
	"context"
	"fmt"

	"github.com/vardius/go-api-boilerplate/pkg/commandbus"
	"github.com/vardius/go-api-boilerplate/pkg/domain"
	apperrors "github.com/vardius/go-api-boilerplate/pkg/errors"
	"github.com/vardius/go-api-boilerplate/pkg/logger"
	messagebus "github.com/vardius/message-bus"
)

// New creates in memory command bus
func New(maxConcurrentCalls int) commandbus.CommandBus {
	return &commandBus{messagebus.New(maxConcurrentCalls)}
}

type commandBus struct {
	messageBus messagebus.MessageBus
}

func (bus *commandBus) Publish(ctx context.Context, command domain.Command) error {
	out := make(chan error, 1)
	defer close(out)

	logger.Debug(ctx, fmt.Sprintf("[CommandBus] Publish: %s %+v", command.GetName(), command))
	bus.messageBus.Publish(command.GetName(), ctx, command, out)

	ctxDoneCh := ctx.Done()
	select {
	case <-ctxDoneCh:
		return apperrors.Wrap(fmt.Errorf("%w: %s", apperrors.ErrTimeout, ctx.Err()))
	case err := <-out:
		if err != nil {
			return apperrors.Wrap(fmt.Errorf("create client failed: %w", err))
		}
		return nil
	}
}

func (bus *commandBus) Subscribe(ctx context.Context, commandName string, fn commandbus.CommandHandler) error {
	logger.Info(nil, fmt.Sprintf("[CommandBus] Subscribe: %s", commandName))

	// unsubscribe all other handlers
	bus.messageBus.Close(commandName)

	return bus.messageBus.Subscribe(commandName, func(ctx context.Context, command domain.Command, out chan<- error) {
		out <- fn(ctx, command)
	})
}

func (bus *commandBus) Unsubscribe(ctx context.Context, commandName string) error {
	logger.Info(nil, fmt.Sprintf("[CommandBus] Unsubscribe: %s", commandName))
	bus.messageBus.Close(commandName)

	return nil
}
