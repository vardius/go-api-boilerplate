package grpc

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/vardius/golog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
)

// ConnectionConfig provides values for gRPC connection configuration
type ConnectionConfig interface {
	GetGrpcConnTime() time.Duration
	GetGrpcConnTimeout() time.Duration
}

// NewConnection provides new grpc connection
func NewConnection(ctx context.Context, host string, port int, cfg ConnectionConfig, logger golog.Logger) *grpc.ClientConn {
	opts := []grpc.DialOption{
		grpc.WithInsecure(),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                cfg.GetGrpcConnTime(),    // send pings every 10 seconds if there is no activity
			Timeout:             cfg.GetGrpcConnTimeout(), // wait 20 second for ping ack before considering the connection dead
			PermitWithoutStream: true,                     // send pings even without active streams
		}),
	}
	conn, err := grpc.DialContext(ctx, fmt.Sprintf("%s:%d", host, port), opts...)
	if err != nil {
		logger.Critical(ctx, "grpc auth conn dial error: %v\n", err)
		os.Exit(1)
	}

	return conn
}
