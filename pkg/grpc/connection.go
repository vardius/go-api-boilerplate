package grpc

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/vardius/go-api-boilerplate/pkg/grpc/middleware"
	"github.com/vardius/go-api-boilerplate/pkg/grpc/middleware/firewall"
	"github.com/vardius/go-api-boilerplate/pkg/logger"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
)

// ConnectionConfig provides values for gRPC connection configuration
type ConnectionConfig struct {
	ConnTime    time.Duration
	ConnTimeout time.Duration
}

// NewConnection provides new grpc connection
func NewConnection(ctx context.Context, host string, port int, cfg ConnectionConfig) *grpc.ClientConn {
	opts := []grpc.DialOption{
		grpc.WithInsecure(),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                cfg.ConnTime,    // send pings every 10 seconds if there is no activity
			Timeout:             cfg.ConnTimeout, // wait 20 second for ping ack before considering the connection dead
			PermitWithoutStream: true,            // send pings even without active streams
		}),
		grpc.WithChainUnaryInterceptor(
			middleware.AppendMetadataToOutgoingUnaryContext(),
			firewall.AppendIdentityToOutgoingUnaryContext(),
			middleware.TransformUnaryIncomingError(),
			middleware.LogOutgoingUnaryRequest(),
		),
		grpc.WithChainStreamInterceptor(
			middleware.AppendMetadataToOutgoingStreamContext(),
			firewall.AppendIdentityToOutgoingStreamContext(),
			middleware.TransformStreamIncomingError(),
			middleware.LogOutgoingStreamRequest(),
		),
	}
	conn, err := grpc.DialContext(ctx, fmt.Sprintf("%s:%d", host, port), opts...)
	if err != nil {
		logger.Critical(ctx, fmt.Sprintf("[gRPC|Client] auth conn dial error: %v", err))
		os.Exit(1)
	}

	return conn
}
