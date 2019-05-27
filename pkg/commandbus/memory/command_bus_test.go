package commandbus

import (
	"context"
	"runtime"
	"testing"
	"time"

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

func TestSubscribePublish(t *testing.T) {
	bus := NewLoggable(runtime.NumCPU(), golog.New("debug"))
	ctx := context.Background()
	c := make(chan error)

	bus.Subscribe("command", func(ctx context.Context, _ bool, out chan<- error) {
		out <- nil
	})

	bus.Publish(ctx, "command", true, c)

	for {
		select {
		case <-ctx.Done():
			t.Fatal(ctx.Err())
			return
		case err := <-c:
			if err != nil {
				t.Error(err)
			}
			return
		}
	}
}

func TestUnsubscribe(t *testing.T) {
	bus := New(runtime.NumCPU())
	c := make(chan error)
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	handler := func(ctx context.Context, _ bool, out chan<- error) {
		t.Fail()
	}

	bus.Subscribe("command", handler)
	bus.Unsubscribe("command", handler)

	bus.Publish(ctx, "command", true, c)

	for {
		select {
		case <-ctx.Done():
			return
		}
	}
}
