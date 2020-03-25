package grpc

import (
	"context"
	"time"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_recovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/status"

	"github.com/vardius/go-api-boilerplate/pkg/grpc/middleware"
	"github.com/vardius/go-api-boilerplate/pkg/log"
)

// ServerConfig provides values for gRPC server configuration
type ServerConfig struct {
	ServerMinTime time.Duration
	ServerTime    time.Duration
	ServerTimeout time.Duration
}

// NewServer provides new grpc server
func NewServer(cfg ServerConfig, logger *log.Logger) *grpc.Server {
	opts := []grpc_recovery.Option{
		grpc_recovery.WithRecoveryHandlerContext(func(ctx context.Context, rec interface{}) (err error) {
			logger.Critical(ctx, "[gRPC|Server] Recovered in %v\n", rec)

			return status.Errorf(codes.Internal, "Recovered in %v\n", rec)
		}),
	}

	server := grpc.NewServer(
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
			MinTime:             cfg.ServerMinTime, // If a client pings more than once every 5 minutes, terminate the connection
			PermitWithoutStream: true,              // Allow pings even when there are no active streams
		}),
		grpc.KeepaliveParams(keepalive.ServerParameters{
			Time:    cfg.ServerTime,    // Ping the client if it is idle for 2 hours to ensure the connection is still active
			Timeout: cfg.ServerTimeout, // Wait 20 second for the ping ack before assuming the connection is dead
		}),
		grpc_middleware.WithUnaryServerChain(
			// firewall.GrantAccessForUnaryRequest("admin"), // TODO: do it per service request
			middleware.LogUnaryRequest(logger),
			middleware.SetMetadataFromUnaryRequest(),
			grpc_recovery.UnaryServerInterceptor(opts...),
		),
		grpc_middleware.WithStreamServerChain(
			// firewall.GrantAccessForStreamRequest("admin"), // TODO: do it per service request
			middleware.LogStreamRequest(logger),
			middleware.SetMetadataFromStreamRequest(),
			grpc_recovery.StreamServerInterceptor(opts...),
		),
	)

	return server
}
