package grpc

import (
	"context"
	"time"

	grpcmiddleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpcrecovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	"github.com/vardius/golog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/status"

	"github.com/vardius/go-api-boilerplate/pkg/grpc/middleware"
	"github.com/vardius/go-api-boilerplate/pkg/grpc/middleware/firewall"
)

// ServerConfig provides values for gRPC server configuration
type ServerConfig struct {
	ServerMinTime time.Duration
	ServerTime    time.Duration
	ServerTimeout time.Duration
}

// NewServer provides new grpc server
func NewServer(cfg ServerConfig, logger golog.Logger, unaryInterceptors []grpc.UnaryServerInterceptor, streamInterceptors []grpc.StreamServerInterceptor) *grpc.Server {
	opts := []grpcrecovery.Option{
		grpcrecovery.WithRecoveryHandlerContext(func(ctx context.Context, rec interface{}) (err error) {
			logger.Critical(ctx, "[gRPC|Server] Recovered in %v", rec)

			return status.Errorf(codes.Internal, "Recovered in %v", rec)
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
		grpcmiddleware.WithUnaryServerChain(
			append([]grpc.UnaryServerInterceptor{
				grpcrecovery.UnaryServerInterceptor(opts...),
				middleware.TransformUnaryIncomingError(),
				middleware.SetMetadataFromUnaryRequest(),
				firewall.SetIdentityFromUnaryRequest(),
				middleware.LogUnaryRequest(logger),
			}, unaryInterceptors...)...,
		),
		grpcmiddleware.WithStreamServerChain(
			append([]grpc.StreamServerInterceptor{
				grpcrecovery.StreamServerInterceptor(opts...),
				middleware.TransformStreamIncomingError(),
				middleware.SetMetadataFromStreamRequest(),
				firewall.SetIdentityFromStreamRequest(),
				middleware.LogStreamRequest(logger),
			}, streamInterceptors...)...,
		),
	)

	return server
}
