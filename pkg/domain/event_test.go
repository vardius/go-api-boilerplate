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
	event, err := NewEvent(uuid.New(), "streamName", 0, rawEventMock{Page: 1, Fruits: []string{"apple", "peach", "pear"}}, nil)
	if err != nil {
		t.Errorf("%s", err)
	}

	mEvent, err2 := MakeEvent(event.Metadata, event.Payload, nil)
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

type invalidRawEventMock struct {
	C chan int
}

func (e invalidRawEventMock) GetType() string {
	return "test.Mock"
}

func TestNewEventInvalidValue(t *testing.T) {
	_, err := NewEvent(uuid.New(), "streamName", 0, invalidRawEventMock{make(chan int)}, nil)
	if err == nil {
		t.Error("Parsing value to json should fail")
	}
}
