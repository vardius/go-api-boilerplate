package middleware

import (
	"context"
	"time"

	"github.com/vardius/golog"
	"google.golang.org/grpc"
	healthproto "google.golang.org/grpc/health/grpc_health_v1"
)

// LogOutgoingUnaryRequest logs client request
//
// https://godoc.org/google.golang.org/grpc#WithUnaryInterceptor
//
// conn, err := grpc.Dial("localhost:5000", grpc.WithUnaryInterceptor(LogOutgoingUnaryRequest()))
func LogOutgoingUnaryRequest(logger golog.Logger) grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		// Skip health check requests
		if _, ok := req.(*healthproto.HealthCheckRequest); ok {
			return invoker(ctx, method, req, reply, cc, opts...)
		}

		now := time.Now()

		logger.Info(ctx, "[gRPC|Client] UnaryRequest Start: %s", method)

		err := invoker(ctx, method, req, reply, cc, opts...)
		if err != nil {
			logger.Warning(ctx, "[gRPC|Client] UnaryRequest End: %s (%s) Err: %v", method, time.Since(now), err)
		} else {
			logger.Info(ctx, "[gRPC|Client] UnaryRequest End: %s (%s)", method, time.Since(now))
		}

		return err
	}
}

// LogOutgoingStreamRequest logs client request
//
// https://godoc.org/google.golang.org/grpc#WithStreamInterceptor
//
// conn, err := grpc.Dial("localhost:5000", grpc.WithStreamInterceptor(LogOutgoingStreamRequest()))
func LogOutgoingStreamRequest(logger golog.Logger) grpc.StreamClientInterceptor {
	return func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
		now := time.Now()

		logger.Info(ctx, "[gRPC|Client] StreamRequest Start: %s", desc.StreamName)

		stream, err := streamer(ctx, desc, cc, method, opts...)

		if err != nil {
			logger.Warning(ctx, "[gRPC|Client] StreamRequest End: %s (%s) Err: %v", desc.StreamName, time.Since(now), err)
		} else {
			logger.Info(ctx, "[gRPC|Client] StreamRequest End: %s (%s)", desc.StreamName, time.Since(now))
		}

		return stream, err
	}
}

// LogStreamRequest returns error if Identity not set within context or user does not have required role
//
// 	https://godoc.org/google.golang.org/grpc#StreamInterceptor
//
// opts := []grpc.ServerOption{
// 	grpc.UnaryInterceptor(LogStreamRequest(logger)),
// }
// s := grpc.NewServer(opts...)
// pb.LogStreamRequest(s, &server{})
func LogStreamRequest(logger golog.Logger) grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		now := time.Now()

		logger.Info(ss.Context(), "[gRPC|Server] StreamRequest Start: %s", info.FullMethod)

		err := handler(srv, ss)
		if err != nil {
			logger.Warning(ss.Context(), "[gRPC|Server] StreamRequest End: %s (%s) Err: %v", info.FullMethod, time.Since(now), err)
		} else {
			logger.Info(ss.Context(), "[gRPC|Server] StreamRequest End: %s (%s)", info.FullMethod, time.Since(now))
		}

		return err
	}
}

// LogUnaryRequest returns error if Identity not set within context or user does not have required role
//
// 	https://godoc.org/google.golang.org/grpc#UnaryInterceptor
//
// opts := []grpc.ServerOption{
// 	grpc.UnaryInterceptor(LogUnaryRequest(logger)),
// }
// s := grpc.NewServer(opts...)
// pb.RegisterGreeterServer(s, &server{})
func LogUnaryRequest(logger golog.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// Skip health check requests
		if _, ok := req.(*healthproto.HealthCheckRequest); ok {
			return handler(ctx, req)
		}

		now := time.Now()

		logger.Info(ctx, "[gRPC|Server] UnaryRequest Start: %s", info.FullMethod)

		resp, err := handler(ctx, req)

		if err != nil {
			logger.Warning(ctx, "[gRPC|Server] UnaryRequest End: %s (%s) Err: %v", info.FullMethod, time.Since(now), err)
		} else {
			logger.Info(ctx, "[gRPC|Server] UnaryRequest End: %s (%s)", info.FullMethod, time.Since(now))
		}

		return resp, err
	}
}
