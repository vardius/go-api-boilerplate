package errors

import (
	"errors"
	"fmt"
	"testing"

	"github.com/vardius/go-api-boilerplate/pkg/application"
)

func TestNew(t *testing.T) {
	err := New("test error")

	if err == nil {
		t.Error("Error should not be nil")
	}
}

func TestWrap(t *testing.T) {
	err := Wrap(fmt.Errorf("test error: %w", application.ErrInternal))

	if err == nil {
		t.Error("Error should not be nil")
	}

	if !errors.Is(err, application.ErrInternal) {
		t.Error("Error is not Internal")
	}
}
