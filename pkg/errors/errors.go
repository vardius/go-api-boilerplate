// Package errors implements functions to manipulate errors.
package errors

import (
	"errors"
	"fmt"
	"strings"

	"github.com/vardius/trace"

	"github.com/vardius/go-api-boilerplate/pkg/application"
)

var (
	DefaultMessageSeparator    = ": "
	DefaultStackTraceSeparator = "\n"
	ErrorFormat                = "%s\n%s"
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
	msg := strings.Join(e.messages(), DefaultMessageSeparator)
	stackTrace := strings.Join(e.StackTrace(), DefaultStackTraceSeparator)

	return fmt.Sprintf(ErrorFormat, msg, stackTrace)
}

func (e *AppError) Unwrap() error {
	return e.err
}

// StackTrace returns the string slice of the error stack traces
func (e *AppError) StackTrace() []string {
	var stack []string

	if e.trace != "" {
		stack = append(stack, e.trace)
	}

	if e.err == nil {
		return stack
	}

	var next *AppError
	if errors.As(e.err, &next) {
		return append(stack, next.StackTrace()...)
	}

	return stack
}

// messages returns the string slice of the error messages
func (e *AppError) messages() []string {
	var messages []string
	if e.err == nil {
		return messages
	}

	if next, ok := e.err.(*AppError); ok {
		messages = append(messages, next.messages()...)
	} else {
		messages = append(messages, e.err.Error())
	}

	return messages
}
