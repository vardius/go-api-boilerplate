package domain

import (
	"bytes"
	"testing"

	"github.com/google/uuid"
)

type mockData struct {
	Page   int      `json:"page"`
	Fruits []string `json:"fruits"`
}

func TestEvent(t *testing.T) {
	event, err := NewEvent(uuid.New(), "streamName", 0, mockData{Page: 1, Fruits: []string{"apple", "peach", "pear"}})
	if err != nil {
		t.Errorf("%s", err)
	}

	mEvent, err2 := MakeEvent(event.Metadata, event.Payload)
	if err2 != nil {
		t.Errorf("%s", err)
	}

	if event.Metadata != mEvent.Metadata {
		t.Error("Events metadata do not match")
	}

	cmp := bytes.Compare(event.Payload, mEvent.Payload)
	if cmp != 0 {
		t.Error("Events payload do not match")
	}
}

func TestNewEventInvalidValue(t *testing.T) {
	_, err := NewEvent(uuid.New(), "streamName", 0, make(chan int))
	if err == nil {
		t.Error("Parsing value to json should fail")
	}
}
