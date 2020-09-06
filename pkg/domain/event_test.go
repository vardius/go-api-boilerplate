package domain

import (
	"bytes"
	"testing"

	"github.com/google/uuid"
)

type rawEventMock struct {
	Page   int      `json:"page"`
	Fruits []string `json:"fruits"`
}

func (e rawEventMock) GetType() string {
	return "test.Mock"
}

func TestEvent(t *testing.T) {
	event, err := NewEventFromRawEvent(uuid.New(), "streamName", 0, rawEventMock{Page: 1, Fruits: []string{"apple", "peach", "pear"}})
	if err != nil {
		t.Error(err)
	}

	mEvent, err2 := NewEventFromPayload(event.StreamID, event.StreamName, event.StreamVersion, event.ID, event.Type, event.OccurredAt, event.Payload)
	if err2 != nil {
		t.Fatal(err)
	}

	if event.ID != mEvent.ID {
		t.Error("Events ID do not match")
	}

	if event.StreamID != mEvent.StreamID {
		t.Error("Events StreamID do not match")
	}

	if event.StreamName != mEvent.StreamName {
		t.Error("Events StreamName do not match")
	}

	if event.StreamVersion != mEvent.StreamVersion {
		t.Error("Events StreamVersion do not match")
	}

	if event.Type != mEvent.Type {
		t.Error("Events Type do not match")
	}

	if event.OccurredAt != mEvent.OccurredAt {
		t.Error("Events OccurredAt do not match")
	}

	cmp := bytes.Compare(event.Payload, mEvent.Payload)
	if cmp != 0 {
		t.Error("Events payload do not match")
	}
}

type invalidRawEventMock struct {
	C chan int
}

func (e invalidRawEventMock) GetType() string {
	return "test.Mock"
}

func TestNewEventInvalidValue(t *testing.T) {
	_, err := NewEventFromRawEvent(uuid.New(), "streamName", 0, invalidRawEventMock{make(chan int)})
	if err == nil {
		t.Error("Parsing value to json should fail")
	}
}
