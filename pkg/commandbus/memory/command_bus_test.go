package memory

import (
	"context"
	"errors"
	"runtime"
	"testing"
	"time"

	"github.com/vardius/go-api-boilerplate/pkg/application"
	"github.com/vardius/go-api-boilerplate/pkg/domain"
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

	bus := New(runtime.NumCPU(), log.New("development"))

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

	bus := New(runtime.NumCPU(), log.New("development"))

	handler := func(ctx context.Context, _ domain.Command) error {
		t.Fail()

		return nil
	}

	bus.Subscribe(ctx, "command", handler)
	bus.Unsubscribe(ctx, "command")

	if err := bus.Publish(ctx, &commandMock{}); err != nil && !errors.Is(err, application.ErrTimeout) {
		t.Error(err)
	}
}
