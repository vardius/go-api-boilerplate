// Package apperr defines the kinds of failure every layer shares.
//
// Code wraps a kind with context as it returns, fmt.Errorf("load user: %w",
// err), and each transport maps kinds to its own status codes in one table:
// httpx for HTTP, connectx for Connect. Wrapped context is for logs. What a
// client may read is set explicitly with New or a ValidationError, so
// internal details never reach a response.
package apperr

import (
	"errors"
	"maps"
	"slices"
	"strings"
)

// The kinds. Test for them with errors.Is.
var (
	ErrInvalid      = errors.New("invalid")
	ErrUnauthorized = errors.New("unauthorized")
	ErrForbidden    = errors.New("forbidden")
	ErrNotFound     = errors.New("not found")
	ErrConflict     = errors.New("conflict")
)

// Error is a kind with a message written for the client.
type Error struct {
	Kind    error
	Message string
}

// New returns an error of kind whose message is safe to show to clients.
func New(kind error, message string) error {
	return &Error{Kind: kind, Message: message}
}

func (e *Error) Error() string { return e.Message }

func (e *Error) Unwrap() error { return e.Kind }

// ValidationError reports which input fields are wrong and why. It is an
// ErrInvalid, and both its fields and its messages are shown to clients.
type ValidationError struct {
	Fields map[string]string
}

// Invalid returns a ValidationError for one field.
func Invalid(field, problem string) error {
	return &ValidationError{Fields: map[string]string{field: problem}}
}

func (e *ValidationError) Error() string {
	var b strings.Builder
	for i, field := range slices.Sorted(maps.Keys(e.Fields)) {
		if i > 0 {
			b.WriteString("; ")
		}
		b.WriteString(field + ": " + e.Fields[field])
	}
	return b.String()
}

func (e *ValidationError) Is(target error) bool { return target == ErrInvalid }

// Message returns the client-safe message carried by err, or "" when err
// carries none.
func Message(err error) string {
	if e, ok := errors.AsType[*Error](err); ok {
		return e.Message
	}
	return ""
}
