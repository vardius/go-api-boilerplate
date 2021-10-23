package memory

import (
	"context"
	"fmt"
	"reflect"
	"sync"

	"github.com/vardius/go-api-boilerplate/pkg/domain"
	apperrors "github.com/vardius/go-api-boilerplate/pkg/errors"
	"github.com/vardius/go-api-boilerplate/pkg/eventbus"
	"github.com/vardius/go-api-boilerplate/pkg/executioncontext"
	"github.com/vardius/go-api-boilerplate/pkg/identity"
	"github.com/vardius/go-api-boilerplate/pkg/logger"
	"github.com/vardius/go-api-boilerplate/pkg/metadata"
	messagebus "github.com/vardius/message-bus"
)

// New creates memory event bus
func New(maxConcurrentCalls int) eventbus.EventBus {
	return &eventBus{
		messageBus: messagebus.New(maxConcurrentCalls),
		handlers:   make(map[string]map[reflect.Value]eventHandler),
	}
}

type eventHandler func(ctx context.Context, event *domain.Event, out chan<- error)

type eventBus struct {
	messageBus messagebus.MessageBus
	mtx        sync.RWMutex
	handlers   map[string]map[reflect.Value]eventHandler
}

func (b *eventBus) Publish(parentCtx context.Context, event *domain.Event) error {
	b.mtx.RLock()
	defer b.mtx.RUnlock()

	handlers, ok := b.handlers[event.Type]
	if !ok {
		return nil
	}

	out := make(chan error, len(handlers))

	flags := executioncontext.FromContext(parentCtx)
	ctx := executioncontext.WithFlag(context.Background(), flags)
	if m, ok := metadata.FromContext(parentCtx); ok {
		ctx = metadata.ContextWithMetadata(ctx, m)
	}
	if i, ok := identity.FromContext(parentCtx); ok {
		ctx = identity.ContextWithIdentity(ctx, i)
	}

	go func() {
		logger.Debug(parentCtx, fmt.Sprintf("[EventBus] Publish: %s %+v", event.Type, event))
		b.messageBus.Publish(event.Type, ctx, event, out)
	}()

	return nil
}

func (b *eventBus) PublishAndAcknowledge(parentCtx context.Context, event *domain.Event) error {
	b.mtx.RLock()
	defer b.mtx.RUnlock()

	handlers, ok := b.handlers[event.Type]
	if !ok {
		return nil
	}

	out := make(chan error, len(handlers))

	flags := executioncontext.FromContext(parentCtx)
	ctx := executioncontext.WithFlag(context.Background(), flags)

	logger.Debug(parentCtx, fmt.Sprintf("[EventBus] PublishAndAcknowledge: %s %+v", event.Type, event))
	b.messageBus.Publish(event.Type, ctx, event, out)

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

		return apperrors.Wrap(err)
	}

	return nil
}

func (b *eventBus) Subscribe(ctx context.Context, eventType string, fn eventbus.EventHandler) error {
	logger.Info(ctx, fmt.Sprintf("[EventBus] Subscribe: %s", eventType))

	handler := func(ctx context.Context, event *domain.Event, out chan<- error) {
		logger.Debug(ctx, fmt.Sprintf("[EventHandler] %s: %s", eventType, event.Payload))

		if err := fn(ctx, event); err != nil {
			logger.Error(ctx, fmt.Sprintf("[EventHandler] %s: %v", eventType, err))
			out <- apperrors.Wrap(err)
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
	logger.Info(ctx, fmt.Sprintf("[EventBus] Unsubscribe: %s", eventType))

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
