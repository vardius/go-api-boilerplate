// Package errors implements functions to manipulate errors.
package errors

import (
	"bytes"
	"fmt"
)

// Application error codes.
const (
	INVALID           = "invalid"            // validation failed
	UNAUTHORIZED      = "unauthorized"       // access denied
	FORBIDDEN         = "forbidden"          // forbidden
	NOTFOUND          = "not_found"          // entity does not exist
	INTERNAL          = "internal"           // internal error
	TEMPORARYDISABLED = "temporary_disabled" // temporary disabled
)

// ErrorCode returns the code of the root error, if available. Otherwise returns INTERNAL.
func ErrorCode(err error) string {
	if err == nil {
		return ""
	} else if e, ok := err.(*appError); ok && e.Code != "" {
		return e.Code
	} else if ok && e.Err != nil {
		return ErrorCode(e.Err)
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
	} else if ok && e.Err != nil {
		return ErrorMessage(e.Err)
	}
	return "An internal error has occurred. Please contact technical support."
}

// New returns an app error that formats as the given text.
func New(message string, code string) error {
	return &appError{
		Code:    code,
		Message: message,
	}
}

// Wrap adds error to the stack
func Wrap(err error, message string, code string) error {
	return &appError{
		Code:    code,
		Message: message,
		Err:     err,
	}
}

type appError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Err     error  `json:"error"`
}

// Error returns the string representation of the error message.
func (e *appError) Error() string {
	var buf bytes.Buffer

	// Print the current error in our stack, if any.
	if e.Code != "" {
		fmt.Fprintf(&buf, "<%s> ", e.Code)
	}

	buf.WriteString(e.Message)

	// If wrapping an error, print its Error() message.
	if e.Err != nil {
		buf.WriteString(e.Err.Error())
	}

	return buf.String()
}
