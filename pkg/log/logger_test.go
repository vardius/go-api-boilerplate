package log

import (
	"testing"
)

func TestNew(t *testing.T) {
	bus := New("development")

	if bus == nil {
		t.Fail()
	}
}
