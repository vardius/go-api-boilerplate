package main

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/caarlos0/env"
	"github.com/vardius/go-api-boilerplate/pkg/auth/infrastructure/proto"
	server "github.com/vardius/go-api-boilerplate/pkg/auth/interfaces/grpc"
	"github.com/vardius/go-api-boilerplate/pkg/common/application/jwt"
	"github.com/vardius/go-api-boilerplate/pkg/common/application/log"
	"github.com/vardius/go-api-boilerplate/pkg/common/application/os/shutdown"
	"google.golang.org/grpc"
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

	grpcServer := grpc.NewServer()
	authServer := server.New(jwtService)

	proto.RegisterAuthenticationServer(grpcServer, authServer)

	lis, err := net.Listen("tcp", fmt.Sprintf("%s:%d", cfg.Host, cfg.Port))
	if err != nil {
		logger.Critical(ctx, "failed to listen %s:%d\n%v\n", cfg.Host, cfg.Port, err)
	} else {
		logger.Info(ctx, "[auth] running at %s:%d\n", cfg.Host, cfg.Port)
	}

	go func() {
		logger.Critical(ctx, "failed to serve: %v\n", grpcServer.Serve(lis))
	}()

	shutdown.GracefulStop(func() {
		logger.Info(ctx, "[auth] shutting down...\n")

		grpcServer.GracefulStop()

		logger.Info(ctx, "[auth] gracefully stopped\n")
	})
}
