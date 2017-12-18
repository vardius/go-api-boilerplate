package domain

import (
	"bytes"
	"testing"

	"github.com/google/uuid"
)

func TestEvent(t *testing.T) {
	event, err := NewEvent(uuid.New(), "streamName", 0, "my data")
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
