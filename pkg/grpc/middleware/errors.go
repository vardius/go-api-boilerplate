package middleware

import (
	"context"
	"errors"
	"fmt"
	apperrors "github.com/vardius/go-api-boilerplate/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// TransformUnaryOutgoingError transforms incoming error into app error
//
// 	https://godoc.org/google.golang.org/grpc#UnaryInterceptor
//
// opts := []grpc.ServerOption{
// 	grpc.UnaryInterceptor(TransformUnaryOutgoingError()),
// }
// s := grpc.NewServer(opts...)
// pb.RegisterGreeterServer(s, &server{})
func TransformUnaryOutgoingError() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		resp, err := handler(ctx, req)
		if err != nil {
			return resp, fromAppError(err)
		}

		return resp, nil
	}
}

// TransformStreamOutgoingError transforms incoming error into app error
//
// 	https://godoc.org/google.golang.org/grpc#StreamInterceptor
//
// opts := []grpc.ServerOption{
// 	grpc.UnaryInterceptor(TransformStreamOutgoingError()),
// }
// s := grpc.NewServer(opts...)
// pb.RegisterGreeterServer(s, &server{})
func TransformStreamOutgoingError() grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		if err := handler(srv, ss); err != nil {
			return fromAppError(err)
		}

		return nil
	}
}

// TransformUnaryIncomingError transforms outgoing apperrors to appropriate status codes
//
// https://godoc.org/google.golang.org/grpc#WithUnaryInterceptor
//
// conn, err := grpc.Dial("localhost:5000", grpc.WithUnaryInterceptor(TransformUnaryIncomingError()))
func TransformUnaryIncomingError() grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		if err := invoker(ctx, method, req, reply, cc, opts...); err != nil {
			return apperrors.Wrap(toAppError(err))
		}

		return nil
	}
}

// TransformStreamIncomingError transforms outgoing apperrors to appropriate status codes
//
// https://godoc.org/google.golang.org/grpc#WithStreamInterceptor
//
// conn, err := grpc.Dial("localhost:5000", grpc.WithStreamInterceptor(TransformStreamIncomingError()))
func TransformStreamIncomingError() grpc.StreamClientInterceptor {
	return func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
		stream, err := streamer(ctx, desc, cc, method, opts...)
		if err != nil {
			return stream, apperrors.Wrap(toAppError(err))
		}

		return stream, err
	}
}

func toAppError(err error) error {
	statusCode := status.Convert(err)
	switch {
	case statusCode.Code() == codes.InvalidArgument:
		err = fmt.Errorf("%w: %s", apperrors.ErrInvalid, err)
	case statusCode.Code() == codes.Unauthenticated:
		err = fmt.Errorf("%w: %s", apperrors.ErrUnauthorized, err)
	case statusCode.Code() == codes.PermissionDenied:
		err = fmt.Errorf("%w: %s", apperrors.ErrForbidden, err)
	case statusCode.Code() == codes.NotFound:
		err = fmt.Errorf("%w: %s", apperrors.ErrNotFound, err)
	case statusCode.Code() == codes.DeadlineExceeded:
		err = fmt.Errorf("%w: %s", apperrors.ErrTimeout, err)
	case statusCode.Code() == codes.Unavailable:
		err = fmt.Errorf("%w: %s", apperrors.ErrTemporaryDisabled, err)
	case statusCode.Code() == codes.Internal:
		err = fmt.Errorf("%w: %s", apperrors.ErrInternal, err)
	case statusCode.Code() == codes.Canceled:
		err = fmt.Errorf("%w: %s", context.Canceled, err)
	default:
		err = fmt.Errorf("%w: %s", apperrors.ErrInternal, err)
	}

	return err
}

func fromAppError(err error) error {
	statusCode := status.Convert(err)
	if statusCode.Code() == codes.Unknown {
		switch {
		case errors.Is(err, apperrors.ErrInvalid):
			return status.Error(codes.InvalidArgument, err.Error())
		case errors.Is(err, apperrors.ErrUnauthorized):
			return status.Error(codes.Unauthenticated, err.Error())
		case errors.Is(err, apperrors.ErrForbidden):
			return status.Error(codes.PermissionDenied, err.Error())
		case errors.Is(err, apperrors.ErrNotFound):
			return status.Error(codes.NotFound, err.Error())
		case errors.Is(err, apperrors.ErrTimeout):
			return status.Error(codes.DeadlineExceeded, err.Error())
		case errors.Is(err, apperrors.ErrTemporaryDisabled):
			return status.Error(codes.Unavailable, err.Error())
		case errors.Is(err, apperrors.ErrInternal):
			return status.Error(codes.Internal, err.Error())
		case errors.Is(err, context.DeadlineExceeded):
			return status.Error(codes.DeadlineExceeded, err.Error())
		case errors.Is(err, context.Canceled):
			return status.Error(codes.Canceled, err.Error())
		default:
			return status.Error(codes.Internal, err.Error())
		}
	}

	return err
}
