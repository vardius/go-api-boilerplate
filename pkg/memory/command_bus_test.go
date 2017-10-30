package memory

import (
	"testing"

	"github.com/vardius/golog"
)

func TestNewCommandBus(t *testing.T) {
	logger := golog.New("debug")
	bus := NewCommandBus(logger)

	if bus == nil {
		t.Fail()
	}
}
