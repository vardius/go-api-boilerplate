package eventbus

import (
	"context"
	"encoding/json"

	"github.com/vardius/pubsub/proto"

	"github.com/vardius/go-api-boilerplate/pkg/domain"
	"github.com/vardius/go-api-boilerplate/pkg/errors"
	"github.com/vardius/go-api-boilerplate/pkg/log"
)

// EventHandler function
type EventHandler func(ctx context.Context, event domain.Event)

// EventBus allow to publis/subscribe to events
type EventBus interface {
	Publish(ctx context.Context, event domain.Event)
	Subscribe(ctx context.Context, eventType string, fn EventHandler) error
}

// New creates pubsub event bus
func New(client proto.MessageBusClient, log *log.Logger) EventBus {
	return &eventBus{client, log}
}

type eventBus struct {
	client proto.MessageBusClient
	logger *log.Logger
}

func (bus *eventBus) Subscribe(ctx context.Context, eventType string, fn EventHandler) error {
	stream, err := bus.client.Subscribe(ctx, &proto.SubscribeRequest{
		Topic: eventType,
	})
	if err != nil {
		bus.logger.Error(ctx, "[EventBus] Subscribe: %v\n", err)
		return errors.Wrap(err, errors.INTERNAL, "EventBus client subscribe error")
	}

	bus.logger.Info(ctx, "[EventBus] Subscribe: %s\n", eventType)

	for {
		resp, err := stream.Recv()
		if err != nil {
			bus.logger.Error(ctx, "[EventBus] Subscribe: stream.Recv error: %v\n", err)
			return errors.Wrap(err, errors.INTERNAL, "EventBus stream recv error")
		}

		var event domain.Event
		err = json.Unmarshal(resp.GetPayload(), &event)
		if err != nil {
			bus.logger.Error(ctx, "[EventBus] Subscribe: Unmarshal error: %v\n", err)
			return errors.Wrap(err, errors.INTERNAL, "EventBus unmarshal error")
		}

		bus.logger.Debug(ctx, "[EventBus] Subscribe: %s %s\n", event.Metadata.Type, event.Payload)

		fn(ctx, event)
	}
}

func (bus *eventBus) Publish(ctx context.Context, event domain.Event) {
	payload, err := json.Marshal(event)
	if err != nil {
		bus.logger.Error(ctx, "[EventBus] Publish: Marshal error: %v\n", err)
		return
	}

	bus.logger.Debug(ctx, "[EventBus] Publish: %s %s\n", event.Metadata.Type, payload)

	bus.client.Publish(ctx, &proto.PublishRequest{
		Topic:   event.Metadata.Type,
		Payload: payload,
	})
}
