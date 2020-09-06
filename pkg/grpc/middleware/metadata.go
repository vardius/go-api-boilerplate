package middleware

import (
	"context"
	"encoding/json"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	mtd "github.com/vardius/go-api-boilerplate/pkg/metadata"
)

const mdMetadataKey = "metadata"

// AppendMetadataToOutgoingUnaryContext appends metadata to outgoing context
//
// https://godoc.org/google.golang.org/grpc#WithUnaryInterceptor
//
// conn, err := grpc.Dial("localhost:5000", grpc.WithUnaryInterceptor(AppendMetadataToOutgoingUnaryContext()))
func AppendMetadataToOutgoingUnaryContext() grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		if m, ok := mtd.FromContext(ctx); ok {
			jsn, err := json.Marshal(m)
			if err != nil {
				return err
			}

			ctx = metadata.AppendToOutgoingContext(ctx, mdMetadataKey, string(jsn))
		}

		return invoker(ctx, method, req, reply, cc, opts...)
	}
}

// AppendMetadataToOutgoingStreamContext appends metadata to outgoing context
//
// https://godoc.org/google.golang.org/grpc#WithStreamInterceptor
//
// conn, err := grpc.Dial("localhost:5000", grpc.WithStreamInterceptor(AppendMetadataToOutgoingStreamContext()))
func AppendMetadataToOutgoingStreamContext() grpc.StreamClientInterceptor {
	return func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
		m, ok := mtd.FromContext(ctx)
		if ok {
			jsn, err := json.Marshal(m)
			if err != nil {
				return nil, err
			}

			ctx = metadata.AppendToOutgoingContext(ctx, mdMetadataKey, string(jsn))
		}

		return streamer(ctx, desc, cc, method, opts...)
	}
}

// SetMetadataFromStreamRequest updates context with metadata
//
// 	https://godoc.org/google.golang.org/grpc#StreamInterceptor
//
// opts := []grpc.ServerOption{
// 	grpc.UnaryInterceptor(SetMetadataFromStreamRequest()),
// }
// s := grpc.NewServer(opts...)
// pb.RegisterGreeterServer(s, &server{})
func SetMetadataFromStreamRequest() grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		if md, ok := metadata.FromIncomingContext(ss.Context()); ok {
			var m mtd.Metadata
			if values := md.Get(mdMetadataKey); len(values) > 0 {
				if err := json.Unmarshal([]byte(values[0]), &m); err != nil {
					return err
				}

				m.Now = time.Now()

				// TODO: update server stream context
				// ctx := mtd.ContextWithMetadata(ss.Context(), &m)
			}
		}

		return handler(srv, ss)
	}
}

// SetMetadataFromUnaryRequest updates context with metadata
//
// 	https://godoc.org/google.golang.org/grpc#UnaryInterceptor
//
// opts := []grpc.ServerOption{
// 	grpc.UnaryInterceptor(SetMetadataFromUnaryRequest()),
// }
// s := grpc.NewServer(opts...)
// pb.RegisterGreeterServer(s, &server{})
func SetMetadataFromUnaryRequest() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		if md, ok := metadata.FromIncomingContext(ctx); ok {
			var m mtd.Metadata
			if values := md.Get(mdMetadataKey); len(values) > 0 {
				if err := json.Unmarshal([]byte(values[0]), &m); err != nil {
					return nil, err
				}

				m.Now = time.Now()

				ctx = mtd.ContextWithMetadata(ctx, &m)
			}
		}

		return handler(ctx, req)
	}
}
