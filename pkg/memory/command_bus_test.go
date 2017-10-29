package memory

import (
	"log"
	"testing"
)

func TestNewCommandBus(t *testing.T) {
	logger := log.New("development")
	bus := NewCommandBus(logger)

	if bus == nil {
		t.Fail()
	}
}
