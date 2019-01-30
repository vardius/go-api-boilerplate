package eventstore

import "testing"

func TestNew(t *testing.T) {
	bus := New()

	if bus == nil {
		t.Fail()
	}
}
