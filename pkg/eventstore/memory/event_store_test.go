package eventstore

import (
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

	e1, err := domain.NewEvent(streamID, streamName, 1, rawEventMock{})
	if err != nil {
		t.Fail()
	}

	e2, err := domain.NewEvent(streamID, streamName, 2, rawEventMock{})
	if err != nil {
		t.Fail()
	}

	store := New()

	if store.Store([]domain.Event{e1, e2}) != nil {
		t.Fail()
	}

	se, err := store.Get(e1.ID)
	if err != nil {
		t.Fail()
	}

	if se.ID != e1.ID {
		t.Fail()
	}

	if len(store.FindAll()) != 2 {
		t.Fail()
	}

	if len(store.GetStream(streamID, streamName)) != 2 {
		t.Fail()
	}
}
