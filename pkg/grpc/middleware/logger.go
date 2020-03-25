package middleware

import (
	"context"
	"encoding/json"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"github.com/vardius/go-api-boilerplate/pkg/log"
	mtd "github.com/vardius/go-api-boilerplate/pkg/metadata"
)

// LogOutgoingUnaryRequest logs client request
//
// https://godoc.org/google.golang.org/grpc#WithUnaryInterceptor
//
// conn, err := grpc.Dial("localhost:5000", grpc.WithUnaryInterceptor(LogOutgoingUnaryRequest()))
func LogOutgoingUnaryRequest(logger *log.Logger) grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		now := time.Now()

		m, ok := mtd.FromContext(ctx)
		if ok {
			now = m.Now
		}

		logger.Info(ctx, "[gRPC|Client] Start\n")

		err := invoker(ctx, method, req, reply, cc, opts...)

		if err != nil {
			logger.Warning(ctx, "[gRPC|Client] End: (%s) Err: %v\n", time.Since(now), err)
		} else {
			logger.Info(ctx, "[gRPC|Client] End: (%s)\n", time.Since(now))
		}

		return err
	}
}

// LogOutgoingStreamRequest logs client request
//
// https://godoc.org/google.golang.org/grpc#WithStreamInterceptor
//
// conn, err := grpc.Dial("localhost:5000", grpc.WithStreamInterceptor(LogOutgoingStreamRequest()))
func LogOutgoingStreamRequest(logger *log.Logger) grpc.StreamClientInterceptor {
	return func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
		now := time.Now()

		m, ok := mtd.FromContext(ctx)
		if ok {
			now = m.Now
		}

		logger.Info(ctx, "[gRPC|Client] Start\n")

		stream, err := streamer(ctx, desc, cc, method, opts...)

		if err != nil {
			logger.Warning(ctx, "[gRPC|Client] End: (%s) Err: %v\n", time.Since(now), err)
		} else {
			logger.Info(ctx, "[gRPC|Client] End: (%s)\n", time.Since(now))
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
func LogStreamRequest(logger *log.Logger) grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		now := time.Now()

		if md, ok := metadata.FromIncomingContext(ss.Context()); ok {
			var m mtd.Metadata
			if values := md.Get(mdMetadataKey); len(values) > 0 {
				if err := json.Unmarshal([]byte(values[0]), &m); err != nil {
					return err
				}

				now = m.Now
			}
		}

		logger.Info(ss.Context(), "[gRPC|Server] Start\n")

		err := handler(srv, ss)

		if err != nil {
			logger.Warning(ss.Context(), "[gRPC|Server] End: (%s) Err: %v\n", time.Since(now), err)
		} else {
			logger.Info(ss.Context(), "[gRPC|Server] End: (%s)\n", time.Since(now))
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
func LogUnaryRequest(logger *log.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		if md, ok := metadata.FromIncomingContext(ctx); ok {
			var m mtd.Metadata
			if values := md.Get(mdMetadataKey); len(values) > 0 {
				if err := json.Unmarshal([]byte(values[0]), &m); err != nil {
					return nil, err
				}

				ctx = mtd.ContextWithMetadata(ctx, &m)
			}
		}
		now := time.Now()

		if md, ok := metadata.FromIncomingContext(ctx); ok {
			var m mtd.Metadata
			if values := md.Get(mdMetadataKey); len(values) > 0 {
				if err := json.Unmarshal([]byte(values[0]), &m); err != nil {
					return nil, err
				}

				now = m.Now
			}
		}

		logger.Info(ctx, "[gRPC|Server] Start\n")

		resp, err := handler(ctx, req)

		if err != nil {
			logger.Warning(ctx, "[gRPC|Server] End: (%s) Err: %v\n", time.Since(now), err)
		} else {
			logger.Info(ctx, "[gRPC|Server] End: (%s)\n", time.Since(now))
		}

		return resp, err
	}
}
