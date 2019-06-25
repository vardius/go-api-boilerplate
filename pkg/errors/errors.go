// Package errors implements functions to manipulate errors.
package errors

import (
	"bytes"
	"fmt"
	"runtime"
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
		frame:   getFrame(3),
	}
}

// Newf returns an app error that formats as the given text.
func Newf(code string, message string, args ...interface{}) error {
	return &appError{
		Code:    code,
		Message: fmt.Sprintf(message, args...),
		frame:   getFrame(3),
	}
}

// Wrap adds error to the stack
func Wrap(err error, code string, message string) error {
	return &appError{
		Code:    code,
		Message: message,
		err:     err,
		frame:   getFrame(3),
	}
}

// Wrapf adds error to the stack
func Wrapf(err error, code string, message string, args ...interface{}) error {
	return &appError{
		Code:    code,
		Message: fmt.Sprintf(message, args...),
		err:     err,
		frame:   getFrame(3),
	}
}

type appError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	err     error
	frame   *runtime.Frame
}

// Error returns the string representation of the error message.
// Calls StackTrace internally.
func (e *appError) Error() string {
	return e.stackTrace(true)
}

// StackTrace returns the string representation of the error stack trace,
// includeFrames appends caller pcs frames to each error message if possible.
func (e *appError) stackTrace(includeFrames bool) string {
	var buf bytes.Buffer

	// Print the current error in our stack, if any.
	if e.Code != "" {
		fmt.Fprintf(&buf, "<%s> ", e.Code)
	}

	fmt.Fprintf(&buf, "%s\n", e.Message)

	if includeFrames && e.frame != nil {
		fmt.Fprintf(&buf, "\t%s:%d\n", e.frame.File, e.frame.Line)
	}

	// If wrapping an error, print its Error() message.
	if e.err != nil {
		buf.WriteString(e.err.Error())
	}

	return buf.String()
}

func getFrame(calldepth int) *runtime.Frame {
	pc, file, line, ok := runtime.Caller(calldepth)
	if !ok {
		return nil
	}

	frame := &runtime.Frame{
		PC:   pc,
		File: file,
		Line: line,
	}

	funcForPc := runtime.FuncForPC(pc)
	if funcForPc != nil {
		frame.Func = funcForPc
		frame.Function = funcForPc.Name()
		frame.Entry = funcForPc.Entry()
	}

	return frame
}
