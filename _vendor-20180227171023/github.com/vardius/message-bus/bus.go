package messagebus

import (
	"fmt"
	"reflect"
	"sync"
)

// MessageBus implements publish/subscribe messaging paradigm
type MessageBus interface {
	Publish(topic string, args ...interface{})
	Subscribe(topic string, fn interface{}) error
	Unsubscribe(topic string, fn interface{}) error
}

type handlersMap map[string][]reflect.Value

type messageBus struct {
	mtx      sync.RWMutex
	handlers handlersMap
}

// Publish arguments to given topic subscribers
func (b *messageBus) Publish(topic string, args ...interface{}) {
	b.mtx.RLock()
	defer b.mtx.RUnlock()

	if hs, ok := b.handlers[topic]; ok {
		rArgs := buildHandlerArgs(args)

		for _, h := range hs {
			go h.Call(rArgs)
		}
	}
}

// Subscribe to a topic
func (b *messageBus) Subscribe(topic string, fn interface{}) error {
	if reflect.TypeOf(fn).Kind() != reflect.Func {
		return fmt.Errorf("%s is not a reflect.Func", reflect.TypeOf(fn))
	}

	b.mtx.Lock()
	defer b.mtx.Unlock()

	b.handlers[topic] = append(b.handlers[topic], reflect.ValueOf(fn))

	return nil
}

// Unsubsriber topic
func (b *messageBus) Unsubscribe(topic string, fn interface{}) error {
	b.mtx.Lock()
	defer b.mtx.Unlock()

	if _, ok := b.handlers[topic]; ok {
		rv := reflect.ValueOf(fn)

		for i, h := range b.handlers[topic] {
			if h == rv {
				b.handlers[topic] = append(b.handlers[topic][:i], b.handlers[topic][i+1:]...)
			}
		}

		return nil
	}

	return fmt.Errorf("Topic %s doesn't exist", topic)
}

func buildHandlerArgs(args []interface{}) []reflect.Value {
	reflectedArgs := make([]reflect.Value, 0)

	for _, arg := range args {
		reflectedArgs = append(reflectedArgs, reflect.ValueOf(arg))
	}

	return reflectedArgs
}

// New creates new MessageBus
func New() MessageBus {
	return &messageBus{
		handlers: make(handlersMap),
	}
}
