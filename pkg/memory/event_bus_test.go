package memory

import (
	"log"
	"testing"
)

func TestNewEventBus(t *testing.T) {
	logger := log.New("development")
	bus := NewEventBus(logger)

	if bus == nil {
		t.Fail()
	}
}
