package errors

import (
	"fmt"
	"strings"
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
	var e error

	e = New("", "")
	if !strings.Contains(e.Error(), fmt.Sprint("\n")) {
		t.Errorf("Error string representation of the error message is invalid: %s", e.Error())
	}

	e = New(INTERNAL, "")
	if !strings.Contains(e.Error(), fmt.Sprintf("<%s> %s\n", INTERNAL, "")) {
		t.Errorf("Error string representation of the error message is invalid: %s", e.Error())
	}

	e = New("", "internal error")
	if !strings.Contains(e.Error(), fmt.Sprintf("%s\n", "internal error")) {
		t.Errorf("Error string representation of the error message is invalid: %s", e.Error())
	}

	e = New(INTERNAL, "internal error")
	if !strings.Contains(e.Error(), fmt.Sprintf("<%s> %s\n", INTERNAL, "internal error")) {
		t.Errorf("Error string representation of the error message is invalid: %s", e.Error())
	}

	e = Wrap(New(INTERNAL, "internal error"), "", "")
	if !strings.Contains(e.Error(), fmt.Sprintf("<%s> %s\n", INTERNAL, "internal error")) {
		t.Errorf("Error string representation of the error message is invalid: %s", e.Error())
	}
}
