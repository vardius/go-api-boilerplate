package memory

import (
	"context"
	"runtime"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/vardius/go-api-boilerplate/pkg/domain"
)

type eventMock struct{}

func (e eventMock) GetType() string {
	return "event"
}

func TestNew(t *testing.T) {
	bus := New(runtime.NumCPU())

	if bus == nil {
		t.Fail()
	}
}

func TestSubscribePublish(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	c := make(chan error, 1)
	bus := New(runtime.NumCPU())

	e, err := domain.NewEventFromRawEvent(uuid.New(), "event", 0, eventMock{})
	if err != nil {
		t.Fatal(err)
	}

	if err := bus.Subscribe(ctx, "event", func(ctx context.Context, event *domain.Event) error {
		c <- nil
		return nil
	}); err != nil {
		t.Error(err)
	}

	if err := bus.Publish(ctx, e); err != nil {
		t.Error(err)
	}

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

	bus := New(runtime.NumCPU())

	e, err := domain.NewEventFromRawEvent(uuid.New(), "event", 0, eventMock{})
	if err != nil {
		t.Fatal(err)
	}

	handler := func(ctx context.Context, event *domain.Event) error {
		t.Fail()

		return nil
	}

	if err := bus.Subscribe(ctx, "event", handler); err != nil {
		t.Error(err)
	}
	if err := bus.Unsubscribe(ctx, "event", handler); err != nil {
		t.Error(err)
	}

	if err := bus.Publish(ctx, e); err != nil {
		t.Error(err)
	}

	<-ctx.Done()
}
