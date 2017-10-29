package memory

import "testing"

func TestNewEventStore(t *testing.T) {
	bus := NewEventStore()

	if bus == nil {
		t.Fail()
	}
}
