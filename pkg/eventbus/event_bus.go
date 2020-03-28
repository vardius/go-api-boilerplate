package eventbus

import (
	"context"
	"encoding/json"

	pubsub_proto "github.com/vardius/pubsub/v2/proto"
	pushpull_proto "github.com/vardius/pushpull/proto"

	"github.com/vardius/go-api-boilerplate/pkg/domain"
	"github.com/vardius/go-api-boilerplate/pkg/errors"
	"github.com/vardius/go-api-boilerplate/pkg/log"
)

// EventHandler function
type EventHandler func(ctx context.Context, event domain.Event)

// EventBus allow to publish/subscribe to events, allow to push/pull events
// when calling Publish, handlers registered with Pull method will not be notified
// when calling Push handlers registered with Subscribe will not be notified
// use Publish/Subscribe if you want every handler to be notified of the event
// use Push/Pull if you want only one handler to pull event from queue
type EventBus interface {
	Publish(ctx context.Context, event domain.Event)
	Subscribe(ctx context.Context, eventType string, fn EventHandler) error

	Pull(ctx context.Context, eventType string, fn EventHandler) error
	Push(ctx context.Context, event domain.Event)
}

// New creates pubsub event bus
func New(pubsub pubsub_proto.PubSubClient, pushpull pushpull_proto.PushPullClient, log *log.Logger) EventBus {
	return &eventBus{pubsub, pushpull, log}
}

type eventBus struct {
	pubsub   pubsub_proto.PubSubClient
	pushpull pushpull_proto.PushPullClient
	logger   *log.Logger
}

// Pull adds worker to pull events from queue,
// pulled even will not be handled by other handlers
func (bus *eventBus) Pull(ctx context.Context, eventType string, fn EventHandler) error {
	stream, err := bus.pushpull.Pull(ctx, &pushpull_proto.PullRequest{
		Topic: eventType,
	})
	if err != nil {
		bus.logger.Error(ctx, "[EventBus] Subscribe: %v\n", err)
		return errors.Wrap(err, errors.INTERNAL, "EventBus pushpull client subscribe error")
	}

	bus.logger.Info(stream.Context(), "[EventBus] Pull: %s\n", eventType)

	for {
		resp, err := stream.Recv()
		if err != nil {
			bus.logger.Error(stream.Context(), "[EventBus] Pull: stream.Recv error: %v\n", err)
			return errors.Wrap(err, errors.INTERNAL, "EventBus stream recv error")
		}

		var event domain.Event
		err = json.Unmarshal(resp.GetPayload(), &event)
		if err != nil {
			bus.logger.Error(stream.Context(), "[EventBus] Pull: Unmarshal error: %v\n", err)
			return errors.Wrap(err, errors.INTERNAL, "EventBus unmarshal error")
		}

		bus.logger.Debug(stream.Context(), "[EventBus] Pull: %s %s\n", event.Metadata.Type, event.Payload)

		fn(stream.Context(), event)
	}
}

// Push pushes event to the queue,
// will be handled by first handler to Pull it from that queue
func (bus *eventBus) Push(ctx context.Context, event domain.Event) {
	payload, err := json.Marshal(event)
	if err != nil {
		bus.logger.Error(ctx, "[EventBus] Push: Marshal error: %v\n", err)
		return
	}

	bus.logger.Debug(ctx, "[EventBus] Push: %s %s\n", event.Metadata.Type, payload)

	if _, err := bus.pushpull.Push(ctx, &pushpull_proto.PushRequest{
		Topic:   event.Metadata.Type,
		Payload: payload,
	}); err != nil {
		bus.logger.Error(ctx, "[EventBus] Push: error: %v\n", err)
		return
	}
}

// Subscribe registers handler to be notified of every event published
func (bus *eventBus) Subscribe(ctx context.Context, eventType string, fn EventHandler) error {
	stream, err := bus.pubsub.Subscribe(ctx, &pubsub_proto.SubscribeRequest{
		Topic: eventType,
	})
	if err != nil {
		bus.logger.Error(ctx, "[EventBus] Subscribe: %v\n", err)
		return errors.Wrap(err, errors.INTERNAL, "EventBus pubsub client subscribe error")
	}

	bus.logger.Info(stream.Context(), "[EventBus] Subscribe: %s\n", eventType)

	for {
		resp, err := stream.Recv()
		if err != nil {
			bus.logger.Error(stream.Context(), "[EventBus] Subscribe: stream.Recv error: %v\n", err)
			return errors.Wrap(err, errors.INTERNAL, "EventBus stream recv error")
		}

		var event domain.Event
		err = json.Unmarshal(resp.GetPayload(), &event)
		if err != nil {
			bus.logger.Error(stream.Context(), "[EventBus] Subscribe: Unmarshal error: %v\n", err)
			return errors.Wrap(err, errors.INTERNAL, "EventBus unmarshal error")
		}

		bus.logger.Debug(stream.Context(), "[EventBus] Subscribe: %s %s\n", event.Metadata.Type, event.Payload)

		fn(stream.Context(), event)
	}
}

// Publish sends event to every client subscribed
func (bus *eventBus) Publish(ctx context.Context, event domain.Event) {
	payload, err := json.Marshal(event)
	if err != nil {
		bus.logger.Error(ctx, "[EventBus] Publish: Marshal error: %v\n", err)
		return
	}

	bus.logger.Debug(ctx, "[EventBus] Publish: %s %s\n", event.Metadata.Type, payload)

	if _, err := bus.pubsub.Publish(ctx, &pubsub_proto.PublishRequest{
		Topic:   event.Metadata.Type,
		Payload: payload,
	}); err != nil {
		bus.logger.Error(ctx, "[EventBus] Publish: error: %v\n", err)
		return
	}
}
