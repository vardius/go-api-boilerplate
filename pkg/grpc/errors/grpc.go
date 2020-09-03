package errors

import (
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/vardius/go-api-boilerplate/pkg/application"
)

func NewGRPCError(err error) error {
	code := codes.Internal

	switch {
	case errors.Is(err, application.ErrInvalid):
		code = codes.InvalidArgument
	case errors.Is(err, application.ErrUnauthorized):
		code = codes.Unauthenticated
	case errors.Is(err, application.ErrForbidden):
		code = codes.PermissionDenied
	case errors.Is(err, application.ErrNotFound):
		code = codes.NotFound
	case errors.Is(err, application.ErrTimeout):
		code = codes.DeadlineExceeded
	case errors.Is(err, application.ErrTemporaryDisabled):
		code = codes.Unavailable
	case errors.Is(err, application.ErrInternal):
		code = codes.Internal
	}

	return status.Error(code, err.Error())
}
