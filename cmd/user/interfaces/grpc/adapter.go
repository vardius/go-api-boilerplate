package grpc

import (
	"context"
	"net"

	"github.com/vardius/go-api-boilerplate/cmd/user/application"
	user_proto "github.com/vardius/go-api-boilerplate/cmd/user/infrastructure/proto"
	grpc_main "google.golang.org/grpc"
	grpc_health "google.golang.org/grpc/health"
	grpc_health_proto "google.golang.org/grpc/health/grpc_health_v1"
)

type grpcAdapter struct {
	address      string
	server       *grpc_main.Server
	healthServer *grpc_health.Server
	userServer   user_proto.UserServiceServer
}

// NewAdapter provides new primary adapter
func NewAdapter(address string, server *grpc_main.Server, healthServer *grpc_health.Server, userServer user_proto.UserServiceServer) application.Adapter {
	return &grpcAdapter{
		address:      address,
		server:       server,
		healthServer: healthServer,
		userServer:   userServer,
	}
}

func (adapter *grpcAdapter) Start(ctx context.Context) error {
	user_proto.RegisterUserServiceServer(adapter.server, adapter.userServer)
	grpc_health_proto.RegisterHealthServer(adapter.server, adapter.healthServer)

	lis, err := net.Listen("tcp", adapter.address)
	if err != nil {
		return err
	}

	adapter.healthServer.SetServingStatus("auth", grpc_health_proto.HealthCheckResponse_SERVING)

	return adapter.server.Serve(lis)
}

func (adapter *grpcAdapter) Stop(ctx context.Context) error {
	adapter.healthServer.SetServingStatus("auth", grpc_health_proto.HealthCheckResponse_NOT_SERVING)

	adapter.server.GracefulStop()

	return nil
}
