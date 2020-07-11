package pubsub

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"sync"
	"time"

	pubsubproto "github.com/vardius/pubsub/v2/proto"

	"github.com/vardius/go-api-boilerplate/pkg/domain"
	"github.com/vardius/go-api-boilerplate/pkg/eventbus"
	"github.com/vardius/go-api-boilerplate/pkg/log"
	"github.com/vardius/go-api-boilerplate/pkg/metadata"
)

// New creates pubsub event bus
func New(handlerTimeout time.Duration, pubsub pubsubproto.PubSubClient, log *log.Logger) eventbus.EventBus {
	return &eventBus{handlerTimeout, pubsub, log, sync.RWMutex{}, make(map[reflect.Value]chan struct{})}
}

type dto struct {
	Event           domain.Event       `json:"event"`
	RequestMetadata *metadata.Metadata `json:"request_metadata,omitempty"`
}

// EventBus allow to publish/subscribe to events, allow to push/pull events
// when calling Publish, handlers registered with Pull method will not be notified
// use Publish/Subscribe if you want every handler to be notified of the event
type eventBus struct {
	handlerTimeout time.Duration
	pubsub         pubsubproto.PubSubClient
	logger         *log.Logger

	mtx                 sync.RWMutex
	unsubscribeChannels map[reflect.Value]chan struct{}
}

// Subscribe registers handler to be notified of every event published
func (bus *eventBus) Subscribe(ctx context.Context, eventType string, fn eventbus.EventHandler) error {
	stream, err := bus.pubsub.Subscribe(ctx, &pubsubproto.SubscribeRequest{
		Topic: eventType,
	})
	if err != nil {
		bus.logger.Error(ctx, "[EventBus] Subscribe: %v\n", err)
		return fmt.Errorf("EventBus pubsub client subscribe error: %w", err)
	}

	bus.logger.Info(stream.Context(), "[EventBus] Subscribe: %s\n", eventType)

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
				bus.logger.Error(stream.Context(), "[EventBus] Subscribe: stream.Recv error: %v\n", err)
				return fmt.Errorf("EventBus stream recv error: %w", err)
			}

			if err := bus.dispatchEvent(resp.GetPayload(), fn); err != nil {
				bus.logger.Error(stream.Context(), "[EventBus] Subscribe: dispatchEvent: %v\n", err)
				return fmt.Errorf("EventBus Subscribe stream dispatchEvent: %w", err)
			}
		}
	}
}

// Publish sends event to every client subscribed
func (bus *eventBus) Publish(ctx context.Context, event domain.Event) {
	o := dto{
		Event: event,
	}

	if m, ok := metadata.FromContext(ctx); ok {
		o.RequestMetadata = m
	}

	payload, err := json.Marshal(o)
	if err != nil {
		bus.logger.Error(ctx, "[EventBus] Publish: Marshal error: %v\n", err)
		return
	}

	bus.logger.Debug(ctx, "[EventBus] Publish: %s %s\n", event.Metadata.Type, payload)

	if _, err := bus.pubsub.Publish(ctx, &pubsubproto.PublishRequest{
		Topic:   event.Metadata.Type,
		Payload: payload,
	}); err != nil {
		bus.logger.Error(ctx, "[EventBus] Publish: error: %v\n", err)
		return
	}
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
