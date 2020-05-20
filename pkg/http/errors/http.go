package errors

import (
	"errors"
	"net/http"

	"github.com/vardius/go-api-boilerplate/pkg/application"
	apperrors "github.com/vardius/go-api-boilerplate/pkg/errors"
)

type HttpError struct {
	Code       int    `json:"code"`
	Message    string `json:"message"`
	RequestID  string `json:"request_id,omitempty"`
	StackTrace string `json:"stack_trace,omitempty"`
}

func NewHttpError(err error) *HttpError {
	code := http.StatusInternalServerError

	switch {
	case errors.Is(err, application.ErrInvalid):
		code = http.StatusBadRequest
	case errors.Is(err, application.ErrUnauthorized):
		code = http.StatusUnauthorized
	case errors.Is(err, application.ErrForbidden):
		code = http.StatusForbidden
	case errors.Is(err, application.ErrNotFound):
		code = http.StatusNotFound
	case errors.Is(err, application.ErrTimeout):
		code = http.StatusRequestTimeout
	case errors.Is(err, application.ErrTemporaryDisabled):
		code = http.StatusServiceUnavailable
	case errors.Is(err, application.ErrInternal):
		code = http.StatusInternalServerError
	}

	var stackTrace string
	var appErr *apperrors.AppError
	if errors.As(err, &appErr) {
		stackTrace, _ = appErr.StackTrace()
	}

	return &HttpError{
		Code:       code,
		StackTrace: stackTrace,
		Message:    http.StatusText(code),
	}
}
