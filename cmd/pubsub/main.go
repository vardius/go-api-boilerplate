package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"time"

	pubsub_config "github.com/vardius/go-api-boilerplate/cmd/pubsub/application/config"
	pubsub_messagebus "github.com/vardius/go-api-boilerplate/cmd/pubsub/application/messagebus"
	pubsub_proto "github.com/vardius/go-api-boilerplate/cmd/pubsub/infrastructure/proto"
	pubsub_grpc "github.com/vardius/go-api-boilerplate/cmd/pubsub/interfaces/grpc"
	"github.com/vardius/go-api-boilerplate/pkg/grpc"
	"github.com/vardius/go-api-boilerplate/pkg/log"
	os_shutdown "github.com/vardius/go-api-boilerplate/pkg/os/shutdown"
	grpc_health "google.golang.org/grpc/health"
	grpc_health_proto "google.golang.org/grpc/health/grpc_health_v1"
)

func main() {
	ctx := context.Background()

	logger := log.New(pubsub_config.Env.Environment)
	bus := pubsub_messagebus.New(pubsub_config.Env.QueueSize, logger)

	grpcServer := grpc.NewServer(logger)
	pubsubServer := pubsub_grpc.NewServer(bus)

	pubsub_proto.RegisterMessageBusServer(grpcServer, pubsubServer)

	healthServer := grpc_health.NewServer()
	healthServer.SetServingStatus("pubsub", grpc_health_proto.HealthCheckResponse_SERVING)

	grpc_health_proto.RegisterHealthServer(grpcServer, healthServer)

	lis, err := net.Listen("tcp", fmt.Sprintf("%s:%d", pubsub_config.Env.Host, pubsub_config.Env.Port))
	if err != nil {
		logger.Critical(ctx, "tcp failed to listen %s:%d\n%v\n", pubsub_config.Env.Host, pubsub_config.Env.Port, err)
		os.Exit(1)
	}

	stop := func() {
		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		logger.Info(ctx, "shutting down...\n")

		grpcServer.GracefulStop()
	}

	go func() {
		logger.Critical(ctx, "failed to serve: %v\n", grpcServer.Serve(lis))
		stop()
		os.Exit(1)
	}()

	logger.Info(ctx, "tcp running at %s:%d\n", pubsub_config.Env.Host, pubsub_config.Env.Port)

	os_shutdown.GracefulStop(stop)
}
