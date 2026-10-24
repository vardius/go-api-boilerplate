package memory

import (
	"context"
	"errors"
	apperrors "github.com/vardius/go-api-boilerplate/pkg/errors"
	"runtime"
	"testing"
	"time"

	"github.com/vardius/go-api-boilerplate/pkg/domain"
)

type commandMock struct{}

func (c *commandMock) GetName() string {
	return "command"
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

	bus := New(runtime.NumCPU())

	bus.Subscribe(ctx, "command", func(ctx context.Context, _ domain.Command) error {
		return nil
	})

	bus.Publish(ctx, &commandMock{})

	if err := bus.Publish(ctx, &commandMock{}); err != nil {
		t.Error(err)
	}
}

func TestUnsubscribe(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	bus := New(runtime.NumCPU())

	handler := func(ctx context.Context, _ domain.Command) error {
		t.Fail()

		return nil
	}

	bus.Subscribe(ctx, "command", handler)
	bus.Unsubscribe(ctx, "command")

	if err := bus.Publish(ctx, &commandMock{}); err != nil && !errors.Is(err, apperrors.ErrTimeout) {
		t.Error(err)
	}
}

func TestPublishSurvivesCancelledContext(t *testing.T) {
	bus := New(1)

	release := make(chan struct{})
	var calls int
	if err := bus.Subscribe(context.Background(), "command", func(ctx context.Context, _ domain.Command) error {
		calls++
		if calls == 1 {
			<-release
		}
		return errors.New("handler result after the caller left")
	}); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := bus.Publish(ctx, &commandMock{}); !errors.Is(err, apperrors.ErrTimeout) {
		t.Fatalf("Publish with a cancelled ctx = %v, want ErrTimeout", err)
	}

	// The first handler now replies to a caller that has gone. Handlers for one
	// command run in order, so once the second call returns the first reply was sent:
	// a reply on a closed channel would have crashed the test binary.
	close(release)
	if err := bus.Publish(context.Background(), &commandMock{}); err == nil {
		t.Fatal("second Publish returned no error, want the handler's error")
	}
}
