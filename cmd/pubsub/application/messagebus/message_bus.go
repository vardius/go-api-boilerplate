package messagebus

import (
	"context"

	"github.com/vardius/golog"
	messagebus "github.com/vardius/message-bus"
)

// Handler function
type Handler interface{}

// Payload of the message
type Payload []byte

// MessageBus allows to subscribe/dispatch messages
type MessageBus interface {
	Publish(topic string, ctx context.Context, payload Payload)
	Subscribe(topic string, fn Handler) error
	Unsubscribe(topic string, fn Handler) error
	Close(topic string)
}

// New creates in memory command bus
func New(maxConcurrentCalls int, log golog.Logger) MessageBus {
	return &loggableMessageBus{messagebus.New(maxConcurrentCalls), log}
}

type loggableMessageBus struct {
	bus    messagebus.MessageBus
	logger golog.Logger
}

func (b *loggableMessageBus) Publish(topic string, ctx context.Context, p Payload) {
	b.logger.Debug(ctx, "[MessageBus|Publish]: %s %s\n", topic, p)
	b.bus.Publish(topic, ctx, p)
}

func (b *loggableMessageBus) Subscribe(topic string, fn Handler) error {
	b.logger.Info(nil, "[MessageBus|Subscribe]: %s\n", topic)
	return b.bus.Subscribe(topic, fn)
}

func (b *loggableMessageBus) Unsubscribe(topic string, fn Handler) error {
	b.logger.Info(nil, "[MessageBus|Unsubscribe]: %s\n", topic)
	return b.bus.Unsubscribe(topic, fn)
}

func (b *loggableMessageBus) Close(topic string) {
	b.logger.Info(nil, "[MessageBus|Close]: %s\n", topic)
	b.bus.Close(topic)
}
