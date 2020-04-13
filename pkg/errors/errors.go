// Package errors implements functions to manipulate errors.
package errors

import (
	"bytes"
	"fmt"

	"github.com/vardius/trace"
)

// Application error codes.
const (
	INVALID           = "invalid"            // validation failed
	UNAUTHORIZED      = "unauthorized"       // access denied
	FORBIDDEN         = "forbidden"          // forbidden
	NOTFOUND          = "not_found"          // entity does not exist
	INTERNAL          = "internal"           // internal error
	TEMPORARYDISABLED = "temporary_disabled" // temporary disabled
	TIMEOUT           = "timeout"            // timeout
)

// ErrorErr returns the code of the root error, if available. Otherwise returns INTERNAL.
func ErrorErr(err error) error {
	if err == nil {
		return nil
	} else if e, ok := err.(*appError); ok && e.err != nil {
		return e.err
	}
	return err
}

// ErrorCode returns the code of the root error, if available. Otherwise returns INTERNAL.
func ErrorCode(err error) string {
	if err == nil {
		return ""
	} else if e, ok := err.(*appError); ok && e.code != "" {
		return e.code
	} else if ok && e.err != nil {
		return ErrorCode(e.err)
	}
	return INTERNAL
}

// ErrorMessage returns the human-readable message of the error, if available.
// Otherwise returns a generic error message.
func ErrorMessage(err error) string {
	if err == nil {
		return ""
	} else if e, ok := err.(*appError); ok && e.message != "" {
		return e.message
	} else if ok && e.err != nil {
		return ErrorMessage(e.err)
	}
	return "An internal error has occurred. Please contact technical support."
}

// New returns an app error that formats as the given text.
func New(code string, message string) error {
	return &appError{
		code:    code,
		message: message,
		trace:   trace.FromParent(1, trace.Lfile|trace.Lline),
	}
}

// Newf returns an app error that formats as the given text.
func Newf(code string, message string, args ...interface{}) error {
	return &appError{
		code:    code,
		message: fmt.Sprintf(message, args...),
		trace:   trace.FromParent(1, trace.Lfile|trace.Lline),
	}
}

// Wrap adds error to the stack
func Wrap(err error, code string, message string) error {
	return &appError{
		code:    code,
		message: message,
		trace:   trace.FromParent(1, trace.Lfile|trace.Lline),
		err:     err,
	}
}

// Wrapf adds error to the stack
func Wrapf(err error, code string, message string, args ...interface{}) error {
	return &appError{
		code:    code,
		message: fmt.Sprintf(message, args...),
		err:     err,
		trace:   trace.FromParent(1, trace.Lfile|trace.Lline),
	}
}

type appError struct {
	code    string
	message string
	trace   string
	err     error
}

// Error returns the string representation of the error message.
// Calls StackTrace internally.
func (e *appError) Error() string {
	s, err := e.stackTrace(true)
	if err != nil {
		// @TODO: handle error
		return s
	}

	return s
}

// StackTrace returns the string representation of the error stack trace,
// includeTrace appends caller pcs frames to each error message if possible.
func (e *appError) stackTrace(includeTrace bool) (string, error) {
	var buf bytes.Buffer

	// Print the current error in our stack, if any.
	if e.code != "" {
		if _, err := fmt.Fprintf(&buf, "<%s> ", e.code); err != nil {
			return "", err
		}
	}

	if _, err := fmt.Fprintf(&buf, "%s\n", e.message); err != nil {
		return "", err
	}

	if includeTrace && e.trace != "" {
		if _, err := fmt.Fprintf(&buf, "\t%s\n", e.trace); err != nil {
			return "", err
		}
	}

	// If wrapping an error, print its Error() message.
	if e.err != nil {
		buf.WriteString(e.err.Error())
	}

	return buf.String(), nil
}
