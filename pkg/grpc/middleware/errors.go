package middleware

import (
	"context"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/vardius/go-api-boilerplate/pkg/application"
	apperrors "github.com/vardius/go-api-boilerplate/pkg/errors"
)

// TransformStreamIncomingError transforms incoming error into app error
//
// 	https://godoc.org/google.golang.org/grpc#StreamInterceptor
//
// opts := []grpc.ServerOption{
// 	grpc.UnaryInterceptor(TransformStreamIncomingError()),
// }
// s := grpc.NewServer(opts...)
// pb.RegisterGreeterServer(s, &server{})
func TransformStreamIncomingError() grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		if err := handler(srv, ss); err != nil {
			return apperrors.Wrap(appError(err))
		}

		return nil
	}
}

// TransformUnaryIncomingError transforms incoming error into app error
//
// 	https://godoc.org/google.golang.org/grpc#UnaryInterceptor
//
// opts := []grpc.ServerOption{
// 	grpc.UnaryInterceptor(TransformUnaryIncomingError()),
// }
// s := grpc.NewServer(opts...)
// pb.RegisterGreeterServer(s, &server{})
func TransformUnaryIncomingError() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		resp, err := handler(ctx, req)
		if err != nil {
			return resp, apperrors.Wrap(appError(err))
		}

		return resp, nil
	}
}

func appError(err error) error {
	errStatus, _ := status.FromError(err)

	switch {
	case errStatus.Code() == codes.InvalidArgument:
		err = fmt.Errorf("%w: %s", application.ErrInvalid, err)
	case errStatus.Code() == codes.Unauthenticated:
		err = fmt.Errorf("%w: %s", application.ErrUnauthorized, err)
	case errStatus.Code() == codes.PermissionDenied:
		err = fmt.Errorf("%w: %s", application.ErrForbidden, err)
	case errStatus.Code() == codes.NotFound:
		err = fmt.Errorf("%w: %s", application.ErrNotFound, err)
	case errStatus.Code() == codes.DeadlineExceeded:
		err = fmt.Errorf("%w: %s", application.ErrTimeout, err)
	case errStatus.Code() == codes.Unavailable:
		err = fmt.Errorf("%w: %s", application.ErrTemporaryDisabled, err)
	default:
		err = fmt.Errorf("%w: %s", application.ErrInternal, err)
	}

	return err
}
