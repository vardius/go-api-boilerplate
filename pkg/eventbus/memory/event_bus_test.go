package eventbus

import (
	"runtime"
	"testing"

	"github.com/vardius/golog"
)

func TestNew(t *testing.T) {
	bus := New(runtime.NumCPU())

	if bus == nil {
		t.Fail()
	}
}

func TestWithLogger(t *testing.T) {
	logger := golog.New("debug")
	parent := New(runtime.NumCPU())
	bus := WithLogger(parent, logger)

	if bus == nil {
		t.Fail()
	}
}

func TestNewLoggable(t *testing.T) {
	logger := golog.New("debug")
	bus := NewLoggable(runtime.NumCPU(), logger)

	if bus == nil {
		t.Fail()
	}
}
