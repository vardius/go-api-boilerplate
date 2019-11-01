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
	logger := golog.New()
	bus := New(runtime.NumCPU(), logger)

	if bus == nil {
		t.Fail()
	}
}

func TestSubscribePublish(t *testing.T) {
	bus := New(runtime.NumCPU(), golog.New())
	ctx := context.Background()
	c := make(chan error, 1)

	bus.Subscribe("command", func(ctx context.Context, _ *commandMock, out chan<- error) {
		out <- nil
	})

	bus.Publish(ctx, &commandMock{}, c)

	ctxDoneCh := ctx.Done()
	for {
		select {
		case <-ctxDoneCh:
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
	bus := New(runtime.NumCPU(), golog.New())
	c := make(chan error, 1)
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	handler := func(ctx context.Context, _ *commandMock, out chan<- error) {
		t.Fail()
	}

	bus.Subscribe("command", handler)
	bus.Unsubscribe("command", handler)

	bus.Publish(ctx, &commandMock{}, c)

	ctxDoneCh := ctx.Done()
	for {
		select {
		case <-ctxDoneCh:
			return
		}
	}
}
