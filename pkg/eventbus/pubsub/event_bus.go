package pubsub

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"sync"
	"time"

	"github.com/vardius/go-api-boilerplate/pkg/domain"
	apperrors "github.com/vardius/go-api-boilerplate/pkg/errors"
	"github.com/vardius/go-api-boilerplate/pkg/eventbus"
	"github.com/vardius/go-api-boilerplate/pkg/logger"
	"github.com/vardius/go-api-boilerplate/pkg/metadata"
	pubsubproto "github.com/vardius/pubsub/v2/proto"
)

// New creates pubsub event bus
func New(handlerTimeout time.Duration, pubsub pubsubproto.PubSubClient) eventbus.EventBus {
	return &eventBus{
		handlerTimeout:      handlerTimeout,
		pubsub:              pubsub,
		unsubscribeChannels: make(map[reflect.Value]chan struct{}),
	}
}

type dto struct {
	Event           *domain.Event      `json:"event"`
	RequestMetadata *metadata.Metadata `json:"request_metadata,omitempty"`
}

// EventBus allow to publish/subscribe to events, allow to push/pull events
// when calling Publish, handlers registered with Pull method will not be notified
// use Publish/Subscribe if you want every handler to be notified of the event
type eventBus struct {
	handlerTimeout time.Duration
	pubsub         pubsubproto.PubSubClient

	mtx                 sync.RWMutex
	unsubscribeChannels map[reflect.Value]chan struct{}
}

// Subscribe registers handler to be notified of every event published
func (b *eventBus) Subscribe(ctx context.Context, eventType string, fn eventbus.EventHandler) error {
	stream, err := b.pubsub.Subscribe(ctx, &pubsubproto.SubscribeRequest{
		Topic: eventType,
	})
	if err != nil {
		return apperrors.Wrap(err)
	}

	logger.Info(stream.Context(), fmt.Sprintf("[EventBus] Subscribe: %s", eventType))

	rv := reflect.ValueOf(fn)
	unsubscribeCh := make(chan struct{}, 1)

	b.mtx.Lock()
	b.unsubscribeChannels[rv] = unsubscribeCh
	b.mtx.Unlock()

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
				return apperrors.Wrap(err)
			}

			if err := b.dispatchEvent(resp.GetPayload(), fn); err != nil {
				return apperrors.Wrap(err)
			}
		}
	}
}

// Publish sends event to every client subscribed
func (b *eventBus) Publish(ctx context.Context, event *domain.Event) error {
	o := dto{
		Event: event,
	}

	if m, ok := metadata.FromContext(ctx); ok {
		o.RequestMetadata = m
	}

	payload, err := json.Marshal(o)
	if err != nil {
		return apperrors.Wrap(err)
	}

	logger.Debug(ctx, fmt.Sprintf("[EventBus] Publish: %s %s", event.Type, string(payload)))

	if _, err := b.pubsub.Publish(ctx, &pubsubproto.PublishRequest{
		Topic:   event.Type,
		Payload: payload,
	}); err != nil {
		return apperrors.Wrap(err)
	}

	return nil
}

func (b *eventBus) PublishAndAcknowledge(ctx context.Context, event *domain.Event) error {
	panic("not implemented")
}

// Unsubscribe will unsubscribe after next event handler because stream.Recv() is blocking
// this method was implemented only to satisfy interface
func (b *eventBus) Unsubscribe(ctx context.Context, eventType string, fn eventbus.EventHandler) error {
	rv := reflect.ValueOf(fn)
	b.mtx.RLock()
	if ch, ok := b.unsubscribeChannels[rv]; ok {
		ch <- struct{}{}
	}
	b.mtx.RUnlock()
	logger.Info(ctx, fmt.Sprintf("[EventBus] Unsubscribe: %s", eventType))
	return nil
}

func (b *eventBus) dispatchEvent(payload []byte, fn eventbus.EventHandler) error {
	ctx, cancel := context.WithTimeout(context.Background(), b.handlerTimeout)
	defer cancel()

	var o dto
	if err := json.Unmarshal(payload, &o); err != nil {
		return apperrors.Wrap(err)
	}

	if o.RequestMetadata != nil {
		ctx = metadata.ContextWithMetadata(ctx, o.RequestMetadata)
	}

	logger.Debug(ctx, fmt.Sprintf("[EventBus] Dispatch Event: %s %s", o.Event.Type, o.Event.Payload))

	return fn(ctx, o.Event)
}
