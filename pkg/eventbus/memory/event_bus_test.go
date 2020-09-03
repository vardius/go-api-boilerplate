package memory

import (
	"context"
	"runtime"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/vardius/go-api-boilerplate/pkg/domain"
	"github.com/vardius/go-api-boilerplate/pkg/log"
)

type eventMock struct{}

func (e eventMock) GetType() string {
	return "event"
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

	e, err := domain.NewEvent(uuid.New(), "event", 0, eventMock{}, nil)
	if err != nil {
		t.Fatal(err)
	}

	_ = bus.Subscribe(ctx, "event", func(ctx context.Context, event domain.Event) error {
		return nil
	})
	_ = bus.Publish(ctx, e)

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

	bus := New(runtime.NumCPU(), log.New("development"))

	e, err := domain.NewEvent(uuid.New(), "event", 0, eventMock{}, nil)
	if err != nil {
		t.Fatal(err)
	}

	handler := func(ctx context.Context, event domain.Event) error {
		t.Fail()

		return nil
	}

	_ = bus.Subscribe(ctx, "event", handler)
	_ = bus.Unsubscribe(ctx, "event", handler)

	_ = bus.Publish(ctx, e)

	ctxDoneCh := ctx.Done()
	for {
		select {
		case <-ctxDoneCh:
			return
		}
	}
}
