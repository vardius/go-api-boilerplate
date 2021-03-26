package eventstore

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/vardius/go-api-boilerplate/pkg/domain"
)

type rawEventMock struct {
	Page   int      `json:"page"`
	Fruits []string `json:"fruits"`
}

func (e rawEventMock) GetType() string {
	return "test.Mock"
}

func TestNew(t *testing.T) {
	store := New()

	if store == nil {
		t.Fail()
	}
}

func TestEventStore(t *testing.T) {
	streamID := uuid.New()
	streamName := "test"

	e1, err := domain.NewEventFromRawEvent(streamID, streamName, 1, rawEventMock{})
	if err != nil {
		t.Fail()
	}

	e2, err := domain.NewEventFromRawEvent(streamID, streamName, 2, rawEventMock{})
	if err != nil {
		t.Fail()
	}

	ctx := context.Background()
	store := New()

	if store.Store(ctx, []*domain.Event{e1, e2}) != nil {
		t.Fail()
	}

	se, err := store.Get(ctx, e1.ID)
	if err != nil {
		t.Fail()
	}

	if se.ID != e1.ID {
		t.Fail()
	}

	es, err := store.FindAll(ctx)
	if err != nil {
		t.Fail()
	}
	if len(es) != 2 {
		t.Fail()
	}

	s, err := store.GetStream(ctx, streamID, streamName)
	if err != nil {
		t.Fail()
	}
	if len(s) != 2 {
		t.Fail()
	}
}
