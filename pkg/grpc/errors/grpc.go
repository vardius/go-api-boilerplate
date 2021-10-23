package errors

import (
	"errors"
	apperrors "github.com/vardius/go-api-boilerplate/pkg/errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func NewGRPCError(err error) error {
	code := codes.Internal

	switch {
	case errors.Is(err, apperrors.ErrInvalid):
		code = codes.InvalidArgument
	case errors.Is(err, apperrors.ErrUnauthorized):
		code = codes.Unauthenticated
	case errors.Is(err, apperrors.ErrForbidden):
		code = codes.PermissionDenied
	case errors.Is(err, apperrors.ErrNotFound):
		code = codes.NotFound
	case errors.Is(err, apperrors.ErrTimeout):
		code = codes.DeadlineExceeded
	case errors.Is(err, apperrors.ErrTemporaryDisabled):
		code = codes.Unavailable
	case errors.Is(err, apperrors.ErrInternal):
		code = codes.Internal
	}

	return status.Error(code, err.Error())
}
