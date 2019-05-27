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

func TestErrorMessage(t *testing.T) {
	msg := "internal error"
	err := New(INTERNAL, msg)

	if ErrorMessage(nil) != "" {
		t.Error("Error message does not match")
	}

	if ErrorMessage(err) != msg {
		t.Error("Error message does not match")
	}
}

func TestErrorCode(t *testing.T) {
	msg := "internal error"
	err := New(INTERNAL, msg)

	if ErrorCode(nil) != "" {
		t.Error("Error message does not match")
	}

	if ErrorCode(err) != INTERNAL {
		t.Error("Error message does not match")
	}
}

func TestError(t *testing.T) {
	if New("", "").Error() != "" {
		t.Error("Error string representation of the error message is invalid")
	}

	if New(INTERNAL, "").Error() == "" {
		t.Error("Error string representation of the error message is invalid")
	}

	if New("", "internal error").Error() == "" {
		t.Error("Error string representation of the error message is invalid")
	}

	if New(INTERNAL, "internal error").Error() == "" {
		t.Error("Error string representation of the error message is invalid")
	}

	if Wrap(New(INTERNAL, "internal error"), "", "").Error() == "" {
		t.Error("Error string representation of the error message is invalid")
	}
}
