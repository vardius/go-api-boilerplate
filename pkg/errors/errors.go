// Package errors implements functions to manipulate errors.
package errors

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/vardius/trace"

	"github.com/vardius/go-api-boilerplate/pkg/application"
)

// New returns an app error that formats as the given text.
func New(message string) error {
	return newAppError(errors.New(message))
}

// Wrap returns an app error.
// If passed value is nil will fallback to internal
func Wrap(err error) error {
	return newAppError(err)
}

func newAppError(err error) error {
	if err == nil {
		err = application.ErrInternal
	}

	return &appError{
		err:   err,
		trace: trace.FromParent(2, trace.Lfile|trace.Lline),
	}
}

type appError struct {
	trace string
	err   error
}

// Error returns the string representation of the error message.
func (e *appError) Error() string {
	return e.err.Error()
}

// Is reports whether any error in err's chain matches target.
func (e *appError) Is(target error) bool {
	if errors.Is(e.err, target) {
		return true
	}

	if next, ok := e.err.(*appError); ok && next != nil {
		return next.Is(target)
	}

	return false
}

// StackTrace returns the string representation of the error stack trace,
// includeTrace appends caller pcs frames to each error message if possible.
func (e *appError) StackTrace() (string, error) {
	var buf bytes.Buffer

	if e.trace != "" {
		if _, err := fmt.Fprintf(&buf, "\t%s\n", e.trace); err != nil {
			return "", err
		}
	}

	if next, ok := e.err.(*appError); ok && next != nil {
		stackTrace, err := next.StackTrace()
		if err != nil {
			return "", err
		}

		buf.WriteString(stackTrace)
	}

	return buf.String(), nil
}
