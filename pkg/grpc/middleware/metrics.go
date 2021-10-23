package middleware

import (
	"context"
	"expvar"
	"google.golang.org/grpc"
)

func CountIncomingStreamRequests() grpc.StreamServerInterceptor {
	counter := expvar.NewInt("grpc_stream_requests")

	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		counter.Add(1)
		return handler(srv, ss)
	}
}

func CountIncomingUnaryRequests() grpc.UnaryServerInterceptor {
	counter := expvar.NewInt("grpc_unary_requests")
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		counter.Add(1)
		return handler(ctx, req)
	}
}
