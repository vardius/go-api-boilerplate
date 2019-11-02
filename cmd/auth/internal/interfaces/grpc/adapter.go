package grpc

import (
	"context"
	"net"

	auth_proto "github.com/vardius/go-api-boilerplate/cmd/auth/proto"
	grpc_main "google.golang.org/grpc"
	grpc_health "google.golang.org/grpc/health"
	grpc_health_proto "google.golang.org/grpc/health/grpc_health_v1"
)

// Adapter is grpc server app adapter
type Adapter struct {
	address      string
	server       *grpc_main.Server
	healthServer *grpc_health.Server
	authServer   auth_proto.AuthenticationServiceServer
}

// NewAdapter provides new primary adapter
func NewAdapter(address string, server *grpc_main.Server, healthServer *grpc_health.Server, authServer auth_proto.AuthenticationServiceServer) *Adapter {
	return &Adapter{
		address:      address,
		server:       server,
		healthServer: healthServer,
		authServer:   authServer,
	}
}

func (adapter *Adapter) Start(ctx context.Context) error {
	auth_proto.RegisterAuthenticationServiceServer(adapter.server, adapter.authServer)
	grpc_health_proto.RegisterHealthServer(adapter.server, adapter.healthServer)

	lis, err := net.Listen("tcp", adapter.address)
	if err != nil {
		return err
	}

	adapter.healthServer.SetServingStatus("auth", grpc_health_proto.HealthCheckResponse_SERVING)

	return adapter.server.Serve(lis)
}

func (adapter *Adapter) Stop(ctx context.Context) error {
	adapter.healthServer.SetServingStatus("auth", grpc_health_proto.HealthCheckResponse_NOT_SERVING)

	adapter.server.GracefulStop()

	return nil
}
