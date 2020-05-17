package errors

import (
	goerrors "errors"
	"fmt"
	"testing"
)

func TestNew(t *testing.T) {
	err := New("test error")

	if err == nil {
		t.Error("Error should not be nil")
	}

	if !goerrors.Is(err, Internal) {
		t.Error("Error is not Internal")
	}
}

func TestWrap(t *testing.T) {
	err := Wrap(fmt.Errorf("test error: %w", Internal))

	if err == nil {
		t.Error("Error should not be nil")
	}

	if !goerrors.Is(err, Internal) {
		t.Error("Error is not Internal")
	}
}

func TestAs(t *testing.T) {
	if !goerrors.Is(AsInvalid(New("test")), Invalid) {
		t.Error("Error is not Invalid")
	}

	if !goerrors.Is(AsUnauthorized(New("test")), Unauthorized) {
		t.Error("Error is not Unauthorized")
	}

	if !goerrors.Is(AsForbidden(New("test")), Forbidden) {
		t.Error("Error is not Forbidden")
	}

	if !goerrors.Is(AsNotfound(New("test")), NotFound) {
		t.Error("Error is not NotFound")
	}

	if !goerrors.Is(AsInternal(New("test")), Internal) {
		t.Error("Error is not Internal")
	}

	if !goerrors.Is(AsTemporaryDisabled(New("test")), TemporaryDisabled) {
		t.Error("Error is not TemporaryDisabled")
	}

	if !goerrors.Is(AsTimeout(New("test")), Timeout) {
		t.Error("Error is not Timeout")
	}
}
