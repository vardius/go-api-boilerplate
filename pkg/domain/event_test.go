package domain

import (
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
	_, err := NewEventFromRawEvent(uuid.New(), "streamName", 0, rawEventMock{Page: 1, Fruits: []string{"apple", "peach", "pear"}})
	if err != nil {
		t.Error(err)
	}
}
