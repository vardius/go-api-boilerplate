package messagebus

import (
	"context"
	"runtime"
	"testing"
	"time"

	"github.com/vardius/golog"
)

func TestNew(t *testing.T) {
	logger := golog.New("debug")
	bus := New(runtime.NumCPU(), logger)

	if bus == nil {
		t.Fail()
	}
}

func TestSubscribePublish(t *testing.T) {
	bus := New(runtime.NumCPU(), golog.New("debug"))
	ctx := context.Background()
	c := make(chan error)

	bus.Subscribe("topic", func(ctx context.Context, _ []byte) {
		c <- nil
	})

	bus.Publish("topic", ctx, []byte("ok"))

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
	bus := New(runtime.NumCPU(), golog.New("debug"))
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	handler := func(ctx context.Context, _ []byte) {
		t.Fail()
	}

	bus.Subscribe("topic", handler)
	bus.Unsubscribe("topic", handler)

	bus.Publish("topic", ctx, []byte("ok"))

	for {
		select {
		case <-ctx.Done():
			return
		}
	}
}
