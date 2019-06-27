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

// ErrorCode returns the code of the root error, if available. Otherwise returns INTERNAL.
func ErrorCode(err error) string {
	if err == nil {
		return ""
	} else if e, ok := err.(*appError); ok && e.Code != "" {
		return e.Code
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
	} else if e, ok := err.(*appError); ok && e.Message != "" {
		return e.Message
	} else if ok && e.err != nil {
		return ErrorMessage(e.err)
	}
	return "An internal error has occurred. Please contact technical support."
}

// New returns an app error that formats as the given text.
func New(code string, message string) error {
	return &appError{
		Code:    code,
		Message: message,
		trace:   trace.FromParent(1, trace.Lfile|trace.Lline),
	}
}

// Newf returns an app error that formats as the given text.
func Newf(code string, message string, args ...interface{}) error {
	return &appError{
		Code:    code,
		Message: fmt.Sprintf(message, args...),
		trace:   trace.FromParent(1, trace.Lfile|trace.Lline),
	}
}

// Wrap adds error to the stack
func Wrap(err error, code string, message string) error {
	return &appError{
		Code:    code,
		Message: message,
		trace:   trace.FromParent(1, trace.Lfile|trace.Lline),
		err:     err,
	}
}

// Wrapf adds error to the stack
func Wrapf(err error, code string, message string, args ...interface{}) error {
	return &appError{
		Code:    code,
		Message: fmt.Sprintf(message, args...),
		err:     err,
		trace:   trace.FromParent(1, trace.Lfile|trace.Lline),
	}
}

type appError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	trace   string
	err     error
}

// Error returns the string representation of the error message.
// Calls StackTrace internally.
func (e *appError) Error() string {
	return e.stackTrace(true)
}

// StackTrace returns the string representation of the error stack trace,
// includeTrace appends caller pcs frames to each error message if possible.
func (e *appError) stackTrace(includeTrace bool) string {
	var buf bytes.Buffer

	// Print the current error in our stack, if any.
	if e.Code != "" {
		fmt.Fprintf(&buf, "<%s> ", e.Code)
	}

	fmt.Fprintf(&buf, "%s\n", e.Message)

	if includeTrace && e.trace != "" {
		fmt.Fprintf(&buf, "\t%s", e.trace)
	}

	// If wrapping an error, print its Error() message.
	if e.err != nil {
		buf.WriteString(e.err.Error())
	}

	return buf.String()
}
