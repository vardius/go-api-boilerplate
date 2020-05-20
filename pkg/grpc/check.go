package grpc

import (
	"context"

	"google.golang.org/grpc"
	healthproto "google.golang.org/grpc/health/grpc_health_v1"
)

// IsConnectionServing checks if GRPC connection status equals HealthCheckResponse_SERVING
func IsConnectionServing(ctx context.Context, service string, conn *grpc.ClientConn) bool {
	resp, err := healthproto.NewHealthClient(conn).Check(ctx, &healthproto.HealthCheckRequest{Service: service})

	return err == nil && resp.GetStatus() == healthproto.HealthCheckResponse_SERVING
}
