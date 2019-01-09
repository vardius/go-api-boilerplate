package main

import (
	"context"
	"fmt"
	"net"
	"runtime"
	"time"

	"github.com/caarlos0/env"
	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_recovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	"github.com/vardius/go-api-boilerplate/pkg/common/application/jwt"
	"github.com/vardius/go-api-boilerplate/pkg/common/application/log"
	"github.com/vardius/go-api-boilerplate/pkg/common/application/os/shutdown"
	commandbus "github.com/vardius/go-api-boilerplate/pkg/common/infrastructure/commandbus/memory"
	eventbus "github.com/vardius/go-api-boilerplate/pkg/common/infrastructure/eventbus/memory"
	eventstore "github.com/vardius/go-api-boilerplate/pkg/common/infrastructure/eventstore/memory"
	"github.com/vardius/go-api-boilerplate/pkg/user/infrastructure/proto"
	server "github.com/vardius/go-api-boilerplate/pkg/user/interfaces/grpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

type config struct {
	Env    string `env:"ENV"    envDefault:"development"`
	Host   string `env:"HOST"   envDefault:"0.0.0.0"`
	Port   int    `env:"PORT"   envDefault:"3002"`
	Secret string `env:"SECRET" envDefault:"secret"`
}

func main() {
	ctx := context.Background()

	cfg := config{}
	env.Parse(&cfg)

	logger := log.New(cfg.Env)

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
	userServer := server.New(
		commandbus.NewLoggable(runtime.NumCPU(), "user", logger),
		eventbus.NewLoggable(runtime.NumCPU(), "user", logger),
		eventstore.New(),
		jwt.New([]byte(cfg.Secret), time.Hour*24),
	)

	proto.RegisterUserServer(grpcServer, userServer)

	healthServer := health.NewHealthServer()
	healthServer.SetServingStatus("user", healthpb.HealthCheckResponse_SERVING)
	healthpb.RegisterHealthServer(grpcServer, healthServer)

	lis, err := net.Listen("tcp", fmt.Sprintf("%s:%d", cfg.Host, cfg.Port))
	if err != nil {
		logger.Critical(ctx, "[user] failed to listen %s:%d\n%v\n", cfg.Host, cfg.Port, err)
	} else {
		logger.Info(ctx, "[user] running at %s:%d\n", cfg.Host, cfg.Port)
	}

	go func() {
		logger.Critical(ctx, "[user] failed to serve: %v\n", grpcServer.Serve(lis))
	}()

	shutdown.GracefulStop(func() {
		logger.Info(ctx, "[user] shutting down...\n")

		grpcServer.GracefulStop()

		logger.Info(ctx, "[user] gracefully stopped\n")
	})
}
