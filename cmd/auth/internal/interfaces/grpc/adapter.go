package grpc

import (
	"context"
	"net"

	grpcmain "google.golang.org/grpc"
	grpchealth "google.golang.org/grpc/health"
	grpchealthproto "google.golang.org/grpc/health/grpc_health_v1"

	authproto "github.com/vardius/go-api-boilerplate/cmd/auth/proto"
)

// Adapter is grpc server app adapter
type Adapter struct {
	address      string
	server       *grpcmain.Server
	healthServer *grpchealth.Server
	authServer   authproto.AuthenticationServiceServer
}

// NewAdapter provides new primary adapter
func NewAdapter(address string, server *grpcmain.Server, healthServer *grpchealth.Server, authServer authproto.AuthenticationServiceServer) *Adapter {
	return &Adapter{
		address:      address,
		server:       server,
		healthServer: healthServer,
		authServer:   authServer,
	}
}

// Start start grpc application adapter
func (adapter *Adapter) Start(ctx context.Context) error {
	authproto.RegisterAuthenticationServiceServer(adapter.server, adapter.authServer)
	grpchealthproto.RegisterHealthServer(adapter.server, adapter.healthServer)

	lis, err := net.Listen("tcp", adapter.address)
	if err != nil {
		return err
	}

	adapter.healthServer.SetServingStatus("auth", grpchealthproto.HealthCheckResponse_SERVING)

	return adapter.server.Serve(lis)
}

// Stop stops grpc application adapter
func (adapter *Adapter) Stop(ctx context.Context) error {
	adapter.healthServer.SetServingStatus("auth", grpchealthproto.HealthCheckResponse_NOT_SERVING)

	adapter.server.GracefulStop()

	return nil
}
