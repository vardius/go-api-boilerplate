// Package errors implements functions to manipulate errors.
package errors

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/vardius/trace"

	"github.com/vardius/go-api-boilerplate/pkg/application"
)

// New returns new app error that formats as the given text.
func New(message string) error {
	return newAppError(errors.New(message))
}

// Wrap returns new app error wrapping target error.
// If passed value is nil will fallback to internal
func Wrap(err error) error {
	return newAppError(err)
}

func newAppError(err error) error {
	if err == nil {
		err = application.ErrInternal
	}

	return &AppError{
		err:   err,
		trace: trace.FromParent(2, trace.Lfile|trace.Lline),
	}
}

type AppError struct {
	trace string
	err   error
}

// Error returns the string representation of the error message.
func (e *AppError) Error() string {
	stack, _ := e.StackTrace()
	return stack
}

func (e *AppError) Unwrap() error {
	return e.err
}

// StackTrace returns the string representation of the error stack trace,
// includeTrace appends caller pcs frames to each error message if possible.
func (e *AppError) StackTrace() (string, error) {
	var buf bytes.Buffer

	if e.trace != "" {
		if _, err := fmt.Fprintf(&buf, "\t%s\n", e.trace); err != nil {
			return "", err
		}
	}

	if e.err == nil {
		return buf.String(), nil
	}

	var next *AppError
	if errors.As(e.err, &next) {
		stackTrace, err := next.StackTrace()
		if err != nil {
			return "", err
		}

		buf.WriteString(stackTrace)
	} else {
		return fmt.Sprintf("%s:\n%s", e.err, buf.String()), nil
	}

	return buf.String(), nil
}
