package eventbus

import (
	"context"
	"runtime"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/vardius/go-api-boilerplate/pkg/domain"
	"github.com/vardius/golog"
)

func TestNew(t *testing.T) {
	bus := New(runtime.NumCPU())

	if bus == nil {
		t.Fail()
	}
}

func TestWithLogger(t *testing.T) {
	parent := New(runtime.NumCPU())
	bus := WithLogger(parent, golog.New("debug"))

	if bus == nil {
		t.Fail()
	}
}

func TestNewLoggable(t *testing.T) {
	bus := NewLoggable(runtime.NumCPU(), golog.New("debug"))

	if bus == nil {
		t.Fail()
	}
}

func TestSubscribePublish(t *testing.T) {
	bus := NewLoggable(runtime.NumCPU(), golog.New("debug"))
	ctx := context.Background()
	c := make(chan domain.Event)

	bus.Subscribe("event", func(ctx context.Context, event domain.Event) {
		c <- event
	})

	e, err := domain.NewEvent(uuid.New(), "test", 1, nil)
	if err != nil {
		t.Fatal(err)
	}

	bus.Publish(ctx, "event", *e)

	for {
		select {
		case <-ctx.Done():
			t.Fatal(ctx.Err())
			return
		case event := <-c:
			if event.ID != e.ID {
				t.Error("Invalid event")
			}
			return
		}
	}
}

func TestUnsubscribe(t *testing.T) {
	bus := NewLoggable(runtime.NumCPU(), golog.New("debug"))
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	handler := func(ctx context.Context, event domain.Event) {
		t.Fail()
	}

	e, err := domain.NewEvent(uuid.New(), "test", 1, nil)
	if err != nil {
		t.Fatal(err)
	}

	bus.Subscribe("event", handler)
	bus.Unsubscribe("event", handler)

	bus.Publish(ctx, "event", *e)

	for {
		select {
		case <-ctx.Done():
			return
		}
	}
}
