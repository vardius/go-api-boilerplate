package grpc

import (
	"context"
	"net"

	grpcmain "google.golang.org/grpc"
	grpchealth "google.golang.org/grpc/health"
	grpchealthproto "google.golang.org/grpc/health/grpc_health_v1"

	userproto "github.com/vardius/go-api-boilerplate/cmd/user/proto"
)

// Adapter is grpc server app adapter
type Adapter struct {
	address      string
	server       *grpcmain.Server
	healthServer *grpchealth.Server
	userServer   userproto.UserServiceServer
}

// NewAdapter provides new primary adapter
func NewAdapter(address string, server *grpcmain.Server, healthServer *grpchealth.Server, userServer userproto.UserServiceServer) *Adapter {
	return &Adapter{
		address:      address,
		server:       server,
		healthServer: healthServer,
		userServer:   userServer,
	}
}

// Start start grpc application adapter
func (adapter *Adapter) Start(ctx context.Context) error {
	userproto.RegisterUserServiceServer(adapter.server, adapter.userServer)
	grpchealthproto.RegisterHealthServer(adapter.server, adapter.healthServer)

	lis, err := net.Listen("tcp", adapter.address)
	if err != nil {
		return err
	}

	adapter.healthServer.SetServingStatus("user", grpchealthproto.HealthCheckResponse_SERVING)

	return adapter.server.Serve(lis)
}

// Stop stops grpc application adapter
func (adapter *Adapter) Stop(ctx context.Context) error {
	adapter.healthServer.SetServingStatus("user", grpchealthproto.HealthCheckResponse_NOT_SERVING)

	adapter.server.GracefulStop()

	return nil
}
