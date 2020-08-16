package pushpull

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"sync"
	"time"

	pushpullproto "github.com/vardius/pushpull/proto"

	"github.com/vardius/go-api-boilerplate/pkg/domain"
	"github.com/vardius/go-api-boilerplate/pkg/errors"
	"github.com/vardius/go-api-boilerplate/pkg/eventbus"
	"github.com/vardius/go-api-boilerplate/pkg/log"
	"github.com/vardius/go-api-boilerplate/pkg/metadata"
)

// New creates pubsub event bus
func New(handlerTimeout time.Duration, client pushpullproto.PushPullClient, log *log.Logger) eventbus.EventBus {
	return &eventBus{handlerTimeout, client, log, sync.RWMutex{}, make(map[reflect.Value]chan struct{})}
}

type dto struct {
	Event           domain.Event       `json:"event"`
	RequestMetadata *metadata.Metadata `json:"request_metadata,omitempty"`
}

// EventBus allow to publish/subscribe to events, allow to push/pull events
// when calling Push handlers registered with Subscribe will not be notified
// use Push/Pull if you want only one handler to pull event from queue
type eventBus struct {
	handlerTimeout time.Duration
	client         pushpullproto.PushPullClient
	logger         *log.Logger

	mtx                 sync.RWMutex
	unsubscribeChannels map[reflect.Value]chan struct{}
}

// Subscribe adds worker to pull events from queue,
// pulled even will not be handled by other handlers
func (bus *eventBus) Subscribe(ctx context.Context, eventType string, fn eventbus.EventHandler) error {
	stream, err := bus.client.Pull(ctx, &pushpullproto.PullRequest{
		Topic: eventType,
	})
	if err != nil {
		bus.logger.Error(ctx, "[EventBus] Pull: %v\n", err)
		return fmt.Errorf("EventBus pushpull client subscribe error: %w", err)
	}

	bus.logger.Info(stream.Context(), "[EventBus] Pull: %s\n", eventType)

	rv := reflect.ValueOf(fn)
	unsubscribeCh := make(chan struct{}, 1)

	bus.mtx.Lock()
	bus.unsubscribeChannels[rv] = unsubscribeCh
	bus.mtx.Unlock()

	ctxDoneCh := ctx.Done()
	for {
		select {
		case <-ctxDoneCh:
			return ctx.Err()
		case <-unsubscribeCh:
			return nil
		default:
			resp, err := stream.Recv()
			if err != nil {
				bus.logger.Error(stream.Context(), "[EventBus] Pull: stream.Recv: %v\n", err)
				return fmt.Errorf("EventBus Pull stream recv: %w", err)
			}

			if err := bus.dispatchEvent(resp.GetPayload(), fn); err != nil {
				bus.logger.Error(stream.Context(), "[EventBus] Pull: dispatchEvent: %v\n", err)
				return fmt.Errorf("EventBus Pull stream dispatchEvent: %w", err)
			}
		}
	}
}

// Publish pushes event to the queue,
// will be handled by first handler to Pull it from that queue
func (bus *eventBus) Publish(ctx context.Context, event domain.Event) error {
	o := dto{
		Event: event,
	}

	if m, ok := metadata.FromContext(ctx); ok {
		o.RequestMetadata = m
	}

	payload, err := json.Marshal(o)
	if err != nil {
		bus.logger.Error(ctx, "[EventBus] Push: Marshal error: %v\n", err)
		return errors.Wrap(err)
	}

	bus.logger.Debug(ctx, "[EventBus] Push: %s %s\n", event.Metadata.Type, payload)

	if _, err := bus.client.Push(ctx, &pushpullproto.PushRequest{
		Topic:   event.Metadata.Type,
		Payload: payload,
	}); err != nil {
		bus.logger.Error(ctx, "[EventBus] Push: error: %v\n", err)
		return errors.Wrap(err)
	}

	return nil
}

// Unsubscribe will unsubscribe after next event handler because stream.Recv() is blocking
// this method was implemented only to satisfy interface
func (bus *eventBus) Unsubscribe(ctx context.Context, eventType string, fn eventbus.EventHandler) error {
	rv := reflect.ValueOf(fn)
	bus.mtx.RLock()
	if ch, ok := bus.unsubscribeChannels[rv]; ok {
		ch <- struct{}{}
	}
	bus.mtx.RUnlock()
	bus.logger.Info(nil, "[EventBus] Unsubscribe: %s\n", eventType)
	return nil
}

func (bus *eventBus) dispatchEvent(payload []byte, fn eventbus.EventHandler) error {
	ctx, cancel := context.WithTimeout(context.Background(), bus.handlerTimeout)
	defer cancel()

	var o dto
	err := json.Unmarshal(payload, &o)
	if err != nil {
		bus.logger.Error(ctx, "[EventBus] Unmarshal error: %v\n", err)
		return fmt.Errorf("EventBus unmarshal error: %w", err)
	}

	if o.RequestMetadata != nil {
		ctx = metadata.ContextWithMetadata(ctx, o.RequestMetadata)
	}

	bus.logger.Debug(ctx, "[EventBus] Dispatch Event: %s %s\n", o.Event.Metadata.Type, o.Event.Payload)

	fn(ctx, o.Event)

	return nil
}
