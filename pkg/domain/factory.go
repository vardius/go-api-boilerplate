package domain

import (
	"fmt"
	"sync"
)

// RawEvent represents raw event that it is aware of its type
type RawEvent interface {
	GetType() string
}

var eventFactories = make(map[string]func() RawEvent)
var eventFactoriesMtx sync.RWMutex

func RegisterEventFactory(eventType string, factory func() RawEvent) error {
	if eventType == "" {
		return fmt.Errorf("invalid event type")
	}

	eventFactoriesMtx.Lock()
	defer eventFactoriesMtx.Unlock()
	if _, ok := eventFactories[eventType]; ok {
		return fmt.Errorf("event for type %s was already registered", eventType)
	}
	eventFactories[eventType] = factory

	return nil
}

func UnregisterEventData(eventType string) error {
	if eventType == "" {
		return fmt.Errorf("invalid event type")
	}

	eventFactoriesMtx.Lock()
	defer eventFactoriesMtx.Unlock()
	if _, ok := eventFactories[eventType]; !ok {
		return fmt.Errorf("event for type %s was not registered", eventType)
	}
	delete(eventFactories, eventType)

	return nil
}

func NewRawEvent(eventType string) (RawEvent, error) {
	eventFactoriesMtx.RLock()
	defer eventFactoriesMtx.RUnlock()
	if factory, ok := eventFactories[eventType]; ok {
		return factory(), nil
	}
	return nil, fmt.Errorf("event for type %s was not registered", eventType)
}
