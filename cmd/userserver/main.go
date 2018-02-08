package main

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/caarlos0/env"
	"github.com/vardius/go-api-boilerplate/internal/userserver"
	"github.com/vardius/go-api-boilerplate/pkg/aws/dynamodb/eventstore"
	"github.com/vardius/go-api-boilerplate/pkg/jwt"
	"github.com/vardius/go-api-boilerplate/pkg/log"
	"github.com/vardius/go-api-boilerplate/pkg/memory/commandbus"
	"github.com/vardius/go-api-boilerplate/pkg/memory/eventbus"
	pb "github.com/vardius/go-api-boilerplate/rpc/domain"
	"google.golang.org/grpc"
	"net"
	"time"
)

type config struct {
	Env         string `env:"ENV"          envDefault:"development"`
	Host        string `env:"HOST"         envDefault:"localhost"`
	Port        int    `env:"PORT"         envDefault:"5001"`
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
	eventBus := eventbus.WithLogger("userserver", eventbus.New(), logger)
	commandBus := commandbus.WithLogger("userserver", commandbus.New(), logger)

	lis, err := net.Listen("tcp", fmt.Sprintf("%s:%d", cfg.Host, cfg.Port))
	if err != nil {
		logger.Critical(ctx, "failed to listen: %v", err)
	} else {
		logger.Info(ctx, "[userserver] running at %s:%d", cfg.Host, cfg.Port)
	}

	grpcServer := grpc.NewServer()
	userServer := userserver.New(
		commandBus,
		eventBus,
		eventStore,
		jwtService,
	)

	pb.RegisterDomainServer(grpcServer, userServer)
	grpcServer.Serve(lis)
}
