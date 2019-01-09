package main

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/caarlos0/env"
	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_recovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	"github.com/vardius/go-api-boilerplate/pkg/auth/infrastructure/proto"
	server "github.com/vardius/go-api-boilerplate/pkg/auth/interfaces/grpc"
	"github.com/vardius/go-api-boilerplate/pkg/common/application/jwt"
	"github.com/vardius/go-api-boilerplate/pkg/common/application/log"
	"github.com/vardius/go-api-boilerplate/pkg/common/application/os/shutdown"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

type config struct {
	Env    string `env:"ENV"    envDefault:"development"`
	Host   string `env:"HOST"   envDefault:"0.0.0.0"`
	Port   int    `env:"PORT"   envDefault:"3001"`
	Secret string `env:"SECRET" envDefault:"secret"`
}

func main() {
	ctx := context.Background()

	cfg := config{}
	env.Parse(&cfg)

	logger := log.New(cfg.Env)
	jwtService := jwt.New([]byte(cfg.Secret), time.Hour*24)

	opts := []grpc_recovery.Option{
		grpc_recovery.WithRecoveryHandlerContext(func(ctx context.Context, rec interface{}) (err error) {
			logger.Critical(ctx, "Recovered in f %v", rec)

			return grpc.Errorf(codes.Internal, "%s", rec)
		}),
	}

	grpcServer := grpc.NewServer(
		grpc_middleware.WithUnaryServerChain(
			grpc_recovery.UnaryServerInterceptor(opts...),
		),
		grpc_middleware.WithStreamServerChain(
			grpc_recovery.StreamServerInterceptor(opts...),
		),
	)
	authServer := server.New(jwtService)

	proto.RegisterAuthenticationServer(grpcServer, authServer)

	healthServer := health.NewHealthServer()
	healthServer.SetServingStatus("auth", healthpb.HealthCheckResponse_SERVING)
	healthpb.RegisterHealthServer(grpcServer, healthServer)

	lis, err := net.Listen("tcp", fmt.Sprintf("%s:%d", cfg.Host, cfg.Port))
	if err != nil {
		logger.Critical(ctx, "[auth] failed to listen %s:%d\n%v\n", cfg.Host, cfg.Port, err)
	} else {
		logger.Info(ctx, "[auth] running at %s:%d\n", cfg.Host, cfg.Port)
	}

	go func() {
		logger.Critical(ctx, "[auth] failed to serve: %v\n", grpcServer.Serve(lis))
	}()

	shutdown.GracefulStop(func() {
		logger.Info(ctx, "[auth] shutting down...\n")

		grpcServer.GracefulStop()

		logger.Info(ctx, "[auth] gracefully stopped\n")
	})
}
