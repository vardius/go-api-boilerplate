package commandbus

import (
	"context"
	"runtime"
	"testing"
	"time"

	"github.com/vardius/golog"
)

type commandMock struct{}

func (c *commandMock) GetName() string {
	return "command"
}

func TestNew(t *testing.T) {
	logger := golog.New(golog.Debug)
	bus := New(runtime.NumCPU(), logger)

	if bus == nil {
		t.Fail()
	}
}

func TestSubscribePublish(t *testing.T) {
	bus := New(runtime.NumCPU(), golog.New(golog.Debug))
	ctx := context.Background()
	c := make(chan error)

	bus.Subscribe("command", func(ctx context.Context, _ *commandMock, out chan<- error) {
		out <- nil
	})

	bus.Publish(ctx, &commandMock{}, c)

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
	bus := New(runtime.NumCPU(), golog.New(golog.Debug))
	c := make(chan error)
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	handler := func(ctx context.Context, _ *commandMock, out chan<- error) {
		t.Fail()
	}

	bus.Subscribe("command", handler)
	bus.Unsubscribe("command", handler)

	bus.Publish(ctx, &commandMock{}, c)

	for {
		select {
		case <-ctx.Done():
			return
		}
	}
}
