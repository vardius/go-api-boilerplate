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
func New(code string, message string) error {
	return &appError{
		Code:    code,
		Message: message,
		pc:      getPCs(),
	}
}

// Newf returns an app error that formats as the given text.
func Newf(code string, message string, args ...interface{}) error {
	return &appError{
		Code:    code,
		Message: fmt.Sprintf(message, args...),
		pc:      getPCs(),
	}
}

// Wrap adds error to the stack
func Wrap(err error, code string, message string) error {
	return &appError{
		Code:    code,
		Message: message,
		Err:     err,
		pc:      getPCs(),
	}
}

// Wrapf adds error to the stack
func Wrapf(err error, code string, message string, args ...interface{}) error {
	return &appError{
		Code:    code,
		Message: fmt.Sprintf(message, args...),
		Err:     err,
		pc:      getPCs(),
	}
}

type appError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Err     error
	pc      []uintptr
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

	if includeFrames && len(e.pc) > 0 {
		frames := runtime.CallersFrames(e.pc)
		// Loop to get frames.
		// A fixed number of pcs can expand to an indefinite number of Frames.
		for {
			frame, more := frames.Next()
			fmt.Fprintf(&buf, "\t%s\n\t%s:%d\n", frame.File, frame.Function, frame.Line)
			if !more {
				break
			}
		}
	}

	// If wrapping an error, print its Error() message.
	if e.Err != nil {
		buf.WriteString(e.Err.Error())
	}

	return buf.String()
}

func getPCs() []uintptr {
	// Ask runtime.Callers for up to 4 pcs, including:
	// - runtime.Callers itself,
	// - package call stack itself
	pc := make([]uintptr, 4)
	n := runtime.Callers(0, pc)

	if n < 4 {
		return pc[:]
	}

	// pass only valid pcs to runtime.CallersFrames
	// exclude irrelevant pcs
	return pc[3:n]
}
