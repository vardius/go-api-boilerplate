package errors

import (
	"errors"
	"fmt"
	"testing"
)

func TestNew(t *testing.T) {
	err := New("test error")

	if err == nil {
		t.Error("Error should not be nil")
	}
}

func TestWrap(t *testing.T) {
	err := Wrap(fmt.Errorf("test error: %w", ErrInternal))

	if err == nil {
		t.Error("Error should not be nil")
	}

	if !errors.Is(err, ErrInternal) {
		t.Error("Error is not Internal")
	}
}
