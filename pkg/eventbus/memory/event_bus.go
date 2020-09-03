package memory

import (
	"context"
	"fmt"
	"reflect"
	"sync"

	messagebus "github.com/vardius/message-bus"

	"github.com/vardius/go-api-boilerplate/pkg/domain"
	"github.com/vardius/go-api-boilerplate/pkg/errors"
	"github.com/vardius/go-api-boilerplate/pkg/eventbus"
	"github.com/vardius/go-api-boilerplate/pkg/executioncontext"
	"github.com/vardius/go-api-boilerplate/pkg/log"
)

// New creates memory event bus
func New(maxConcurrentCalls int, log *log.Logger) eventbus.EventBus {
	return &eventBus{
		messageBus: messagebus.New(maxConcurrentCalls),
		logger:     log,
		handlers:   make(map[string]map[reflect.Value]eventHandler),
	}
}

type eventHandler func(ctx context.Context, event domain.Event, out chan<- error)

type eventBus struct {
	messageBus messagebus.MessageBus
	logger     *log.Logger
	mtx        sync.RWMutex
	handlers   map[string]map[reflect.Value]eventHandler
}

func (b *eventBus) Publish(parentCtx context.Context, event domain.Event) error {
	b.mtx.RLock()
	defer b.mtx.RUnlock()

	handlers, ok := b.handlers[event.Metadata.Type]
	if !ok {
		return nil
	}

	out := make(chan error, len(handlers))

	flags := executioncontext.FromContext(parentCtx)
	ctx := executioncontext.WithFlag(context.Background(), flags)

	go func() {
		b.logger.Debug(parentCtx, "[EventBus] Publish: %s %+v\n", event.Metadata.Type, event)
		b.messageBus.Publish(event.Metadata.Type, ctx, event, out)
	}()

	return nil
}

func (b *eventBus) PublishAndAcknowledge(parentCtx context.Context, event domain.Event) error {
	b.mtx.RLock()
	defer b.mtx.RUnlock()

	handlers, ok := b.handlers[event.Metadata.Type]
	if !ok {
		return nil
	}

	out := make(chan error, len(handlers))

	flags := executioncontext.FromContext(parentCtx)
	ctx := executioncontext.WithFlag(context.Background(), flags)

	b.logger.Debug(parentCtx, "[EventBus] PublishAndAcknowledge: %s %+v\n", event.Metadata.Type, event)
	b.messageBus.Publish(event.Metadata.Type, ctx, event, out)

	var errs []error

	for j := 1; j <= len(handlers); j++ {
		if err := <-out; err != nil {
			errs = append(errs, err)
		}
	}
	close(out)

	if len(errs) > 0 {
		var err error
		for _, handlerErr := range errs {
			err = fmt.Errorf("%v\n%v", err, handlerErr)
		}

		return errors.Wrap(err)
	}

	return nil
}

func (b *eventBus) Subscribe(ctx context.Context, eventType string, fn eventbus.EventHandler) error {
	b.logger.Info(ctx, "[EventBus] Subscribe: %s\n", eventType)

	handler := func(ctx context.Context, event domain.Event, out chan<- error) {
		b.logger.Debug(ctx, "[EventHandler] %s: %s\n", eventType, event.Payload)

		if err := fn(ctx, event); err != nil {
			b.logger.Error(ctx, "[EventHandler] %s: %v\n", eventType, err)
			out <- errors.Wrap(err)
		} else {
			out <- nil
		}
	}

	rv := reflect.ValueOf(fn)

	b.mtx.Lock()
	defer b.mtx.Unlock()

	if _, ok := b.handlers[eventType]; !ok {
		b.handlers[eventType] = make(map[reflect.Value]eventHandler)
	}

	b.handlers[eventType][rv] = handler

	return b.messageBus.Subscribe(eventType, handler)
}

func (b *eventBus) Unsubscribe(ctx context.Context, eventType string, fn eventbus.EventHandler) error {
	b.logger.Info(ctx, "[EventBus] Unsubscribe: %s\n", eventType)

	rv := reflect.ValueOf(fn)

	b.mtx.Lock()
	defer b.mtx.Unlock()

	if topicHandlers, ok := b.handlers[eventType]; ok {
		if handler, ok := topicHandlers[rv]; ok {
			delete(topicHandlers, rv)
			if len(topicHandlers) == 0 {
				delete(b.handlers, eventType)
			}

			return b.messageBus.Unsubscribe(eventType, handler)
		}
	}

	return nil
}
