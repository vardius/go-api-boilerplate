package pushpull

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
	pushpullproto "github.com/vardius/pushpull/proto"
)

// New creates pubsub event bus
func New(handlerTimeout time.Duration, client pushpullproto.PushPullClient) eventbus.EventBus {
	return &eventBus{
		handlerTimeout:      handlerTimeout,
		client:              client,
		unsubscribeChannels: make(map[reflect.Value]chan struct{}),
	}
}

type dto struct {
	Event           *domain.Event      `json:"event"`
	RequestMetadata *metadata.Metadata `json:"request_metadata,omitempty"`
}

// EventBus allow to publish/subscribe to events, allow to push/pull events
// when calling Push handlers registered with Subscribe will not be notified
// use Push/Pull if you want only one handler to pull event from queue
type eventBus struct {
	handlerTimeout time.Duration
	client         pushpullproto.PushPullClient

	mtx                 sync.RWMutex
	unsubscribeChannels map[reflect.Value]chan struct{}
}

// Subscribe adds worker to pull events from queue,
// pulled even will not be handled by other handlers
func (b *eventBus) Subscribe(ctx context.Context, eventType string, fn eventbus.EventHandler) error {
	stream, err := b.client.Pull(ctx, &pushpullproto.PullRequest{
		Topic: eventType,
	})
	if err != nil {
		return apperrors.Wrap(err)
	}

	logger.Info(stream.Context(), fmt.Sprintf("[EventBus] Pull: %s", eventType))

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

// Publish pushes event to the queue,
// will be handled by first handler to Pull it from that queue
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

	logger.Debug(ctx, fmt.Sprintf("[EventBus] Push: %s %s", event.Type, payload))

	if _, err := b.client.Push(ctx, &pushpullproto.PushRequest{
		Topic:   event.Type,
		Payload: payload,
	}); err != nil {
		return apperrors.Wrap(err)
	}

	return nil
}

func (b *eventBus) PublishAndAcknowledge(parentCtx context.Context, event *domain.Event) error {
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
