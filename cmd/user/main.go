package main

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/caarlos0/env"
	"github.com/vardius/go-api-boilerplate/pkg/common/aws/dynamodb/eventstore"
	"github.com/vardius/go-api-boilerplate/pkg/common/jwt"
	"github.com/vardius/go-api-boilerplate/pkg/common/log"
	"github.com/vardius/go-api-boilerplate/pkg/common/memory/commandbus"
	"github.com/vardius/go-api-boilerplate/pkg/common/memory/eventbus"
	"github.com/vardius/go-api-boilerplate/pkg/common/os/shutdown"
	"github.com/vardius/go-api-boilerplate/pkg/common/proto"
	server "github.com/vardius/go-api-boilerplate/pkg/user/interfaces/grpc"
	"google.golang.org/grpc"
)

type config struct {
	Env         string `env:"ENV"          envDefault:"development"`
	Host        string `env:"HOST"         envDefault:"localhost"`
	Port        int    `env:"PORT"         envDefault:"3001"`
	Secret      string `env:"SECRET"       envDefault:"secret"`
	AwsRegion   string `env:"AWS_REGION"   envDefault:"us-east-1"`
	AwsEndpoint string `env:"AWS_ENDPOINT" envDefault:"http://localhost:4569"`
}

func main() {
	ctx := context.Background()

	cfg := config{}
	env.Parse(&cfg)

	awsConfig := &aws.Config{
		Region:   aws.String(cfg.AwsRegion),
		Endpoint: aws.String(cfg.AwsEndpoint),
	}

	logger := log.New(cfg.Env)
	jwtService := jwt.New([]byte(cfg.Secret), time.Hour*24)
	eventStore := eventstore.New("events", awsConfig)
	eventBus := eventbus.WithLogger("user", eventbus.New(), logger)
	commandBus := commandbus.WithLogger("user", commandbus.New(), logger)

	grpcServer := grpc.NewServer()
	userServer := server.New(
		commandBus,
		eventBus,
		eventStore,
		jwtService,
	)

	proto.RegisterDomainServer(grpcServer, userServer)

	lis, err := net.Listen("tcp", fmt.Sprintf("%s:%d", cfg.Host, cfg.Port))
	if err != nil {
		logger.Critical(ctx, "failed to listen: %v\n", err)
	} else {
		logger.Info(ctx, "[user] running at %s:%d\n", cfg.Host, cfg.Port)
	}

	go func() {
		logger.Critical(ctx, "failed to serve: %v\n", grpcServer.Serve(lis))
	}()

	shutdown.GracefulStop(func() {
		logger.Info(ctx, "[user] shutting down...\n")

		grpcServer.GracefulStop()

		logger.Info(ctx, "[user] gracefully stopped\n")
	})
}
