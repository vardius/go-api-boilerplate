package grpc

import (
	"context"

	"google.golang.org/grpc"
	health_proto "google.golang.org/grpc/health/grpc_health_v1"
)

// IsConnectionServing checks if GRPC connection status equals HealthCheckResponse_SERVING
func IsConnectionServing(service string, conn *grpc.ClientConn) bool {
	resp, err := health_proto.NewHealthClient(conn).Check(context.Background(), &health_proto.HealthCheckRequest{Service: service})

	return err == nil && resp.GetStatus() == health_proto.HealthCheckResponse_SERVING
}
