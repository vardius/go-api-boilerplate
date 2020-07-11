package memory

import (
	"context"
	"runtime"
	"testing"
	"time"

	"github.com/vardius/go-api-boilerplate/pkg/log"
)

type commandMock struct{}

func (c *commandMock) GetName() string {
	return "command"
}

func TestNew(t *testing.T) {
	logger := log.New("dev")
	bus := New(runtime.NumCPU(), logger)

	if bus == nil {
		t.Fail()
	}
}

func TestSubscribePublish(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	c := make(chan error, 1)
	bus := New(runtime.NumCPU(), log.New("development"))

	bus.Subscribe(ctx, "command", func(ctx context.Context, _ *commandMock, out chan<- error) {
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
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	c := make(chan error, 1)
	bus := New(runtime.NumCPU(), log.New("development"))

	handler := func(ctx context.Context, _ *commandMock, out chan<- error) {
		t.Fail()
	}

	bus.Subscribe(ctx, "command", handler)
	bus.Unsubscribe(ctx, "command", handler)

	bus.Publish(ctx, &commandMock{}, c)

	ctxDoneCh := ctx.Done()
	for {
		select {
		case <-ctxDoneCh:
			return
		}
	}
}
