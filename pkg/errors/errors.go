// Package errors implements functions to manipulate errors.
package errors

import (
	"bytes"
	goerrors "errors"
	"fmt"

	"github.com/vardius/trace"
)

// Application error.
var (
	Invalid           = goerrors.New("validation failed")
	Unauthorized      = goerrors.New("access denied")
	Forbidden         = goerrors.New("forbidden")
	NotFound          = goerrors.New("not found")
	Internal          = goerrors.New("internal system error")
	TemporaryDisabled = goerrors.New("temporary disabled")
	Timeout           = goerrors.New("timeout")
)

// New returns an app error that formats as the given text.
func New(message string) error {
	return newAppError(goerrors.New(message))
}

// Wrap returns an app error.
// If passed value is nil will fallback to internal
func Wrap(err error) error {
	return newAppError(err)
}

// AsInvalid wraps error as Internal error
func AsInvalid(err error) error {
	return newAppError(fmt.Errorf("%w: %s", Invalid, err))
}

// AsUnauthorized wraps error as Unauthorized error
func AsUnauthorized(err error) error {
	return newAppError(fmt.Errorf("%w: %s", Unauthorized, err))
}

// AsForbidden wraps error as Forbidden error
func AsForbidden(err error) error {
	return newAppError(fmt.Errorf("%w: %s", Forbidden, err))
}

// AsNotfound wraps error as NotFound error
func AsNotfound(err error) error {
	return newAppError(fmt.Errorf("%w: %s", NotFound, err))
}

// AsInternal wraps error as Internal error
func AsInternal(err error) error {
	return newAppError(fmt.Errorf("%w: %s", Internal, err))
}

// AsTemporaryDisabled wraps error as TemporaryDisabled error
func AsTemporaryDisabled(err error) error {
	return newAppError(fmt.Errorf("%w: %s", TemporaryDisabled, err))
}

// AsTimeout wraps error as Timeout error
func AsTimeout(err error) error {
	return newAppError(fmt.Errorf("%w: %s", Timeout, err))
}

func newAppError(err error) error {
	if err == nil {
		err = Internal
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
	return goerrors.Is(e.err, target)
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
