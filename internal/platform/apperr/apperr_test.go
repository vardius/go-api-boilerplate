package apperr_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/vardius/go-api-boilerplate/internal/platform/apperr"
)

func TestMessageSurvivesWrapping(t *testing.T) {
	err := fmt.Errorf("load user 42: %w", apperr.New(apperr.ErrNotFound, "No account uses this email."))
	if !errors.Is(err, apperr.ErrNotFound) {
		t.Error("lost the kind")
	}
	if got := apperr.Message(err); got != "No account uses this email." {
		t.Errorf("Message = %q", got)
	}
	if got := apperr.Message(fmt.Errorf("query: %w", apperr.ErrConflict)); got != "" {
		t.Errorf("Message of a bare kind = %q, want empty", got)
	}
}

func TestValidationError(t *testing.T) {
	err := fmt.Errorf("decode: %w", &apperr.ValidationError{Fields: map[string]string{"email": "is not an address", "code": "is required"}})
	if !errors.Is(err, apperr.ErrInvalid) {
		t.Error("a ValidationError is not ErrInvalid")
	}
	if got := err.Error(); got != "decode: code: is required; email: is not an address" {
		t.Errorf("Error = %q", got)
	}
	if v, ok := errors.AsType[*apperr.ValidationError](apperr.Invalid("email", "is required")); !ok || v.Fields["email"] != "is required" {
		t.Errorf("Invalid = %+v", v)
	}
}
