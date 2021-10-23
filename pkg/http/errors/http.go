package errors

import (
	"context"
	"errors"
	apperrors "github.com/vardius/go-api-boilerplate/pkg/errors"
	"net/http"

	mtd "github.com/vardius/go-api-boilerplate/pkg/metadata"
)

type HttpError struct {
	Code      int    `json:"code"`
	Message   string `json:"message"`
	RequestID string `json:"request_id,omitempty"`
}

func NewHttpError(ctx context.Context, err error) *HttpError {
	code := http.StatusInternalServerError

	switch {
	case errors.Is(err, apperrors.ErrInvalid):
		code = http.StatusBadRequest
	case errors.Is(err, apperrors.ErrUnauthorized):
		code = http.StatusUnauthorized
	case errors.Is(err, apperrors.ErrForbidden):
		code = http.StatusForbidden
	case errors.Is(err, apperrors.ErrNotFound):
		code = http.StatusNotFound
	case errors.Is(err, apperrors.ErrTimeout):
		code = http.StatusRequestTimeout
	case errors.Is(err, apperrors.ErrTemporaryDisabled):
		code = http.StatusServiceUnavailable
	case errors.Is(err, apperrors.ErrInternal):
		code = http.StatusInternalServerError
	}

	httpError := &HttpError{
		Code:    code,
		Message: http.StatusText(code),
	}

	if m, ok := mtd.FromContext(ctx); ok {
		httpError.RequestID = m.TraceID
		m.Err = err
	}

	return httpError
}
