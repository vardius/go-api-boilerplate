package grpc

import (
	"context"
	"fmt"
	"os"

	"github.com/vardius/golog"
	"google.golang.org/grpc"
)

// NewConnection provides new grpc connection
func NewConnection(ctx context.Context, host string, port int, logger golog.Logger) *grpc.ClientConn {
	conn, err := grpc.DialContext(ctx, fmt.Sprintf("%s:%d", host, port), grpc.WithInsecure())
	if err != nil {
		logger.Critical(ctx, "grpc auth conn dial error: %v\n", err)
		os.Exit(1)
	}

	return conn
}
