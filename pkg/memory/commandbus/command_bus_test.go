package commandbus

import (
	"testing"

	"github.com/vardius/golog"
)

func TestNew(t *testing.T) {
	bus := New()

	if bus == nil {
		t.Fail()
	}
}

func TestWithLogger(t *testing.T) {
	logger := golog.New("debug")
	parent := New()
	bus := WithLogger(parent, logger)

	if bus == nil {
		t.Fail()
	}
}
