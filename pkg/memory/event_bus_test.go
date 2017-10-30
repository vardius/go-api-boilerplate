package memory

import (
	"testing"

	"github.com/vardius/golog"
)

func TestNewEventBus(t *testing.T) {
	logger := golog.New("debug")
	bus := NewEventBus(logger)

	if bus == nil {
		t.Fail()
	}
}
