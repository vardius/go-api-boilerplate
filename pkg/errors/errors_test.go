package errors

import (
	"testing"
)

func TestNew(t *testing.T) {
	msg := "internal error"
	err := New(INTERNAL, msg)

	if err == nil {
		t.Error("Error should not be nil")
	}

	if ErrorCode(err) != INTERNAL {
		t.Error("Error code does not match")
	}

	if ErrorMessage(err) != msg {
		t.Error("Error message does not match")
	}
}

func TestWrap(t *testing.T) {
	subMsg := "invalid error"
	subErr := New(INVALID, subMsg)

	msg := "internal error"
	err := Wrap(subErr, INTERNAL, msg)

	if err == nil {
		t.Error("Error should not be nil")
	}

	if ErrorCode(err) != INTERNAL {
		t.Error("Error code does not match")
	}

	if ErrorMessage(err) != msg {
		t.Error("Error message does not match")
	}
}
