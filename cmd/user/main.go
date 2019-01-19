package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"runtime"
	"time"

	"github.com/caarlos0/env"
	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_recovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	"github.com/rs/cors"
	"github.com/vardius/go-api-boilerplate/pkg/common/application/http/response"
	"github.com/vardius/go-api-boilerplate/pkg/common/application/jwt"
	"github.com/vardius/go-api-boilerplate/pkg/common/application/log"
	"github.com/vardius/go-api-boilerplate/pkg/common/application/os/shutdown"
	"github.com/vardius/go-api-boilerplate/pkg/common/application/recovery"
	"github.com/vardius/go-api-boilerplate/pkg/common/application/security/authenticator"
	commandbus "github.com/vardius/go-api-boilerplate/pkg/common/infrastructure/commandbus/memory"
	eventbus "github.com/vardius/go-api-boilerplate/pkg/common/infrastructure/eventbus/memory"
	eventstore "github.com/vardius/go-api-boilerplate/pkg/common/infrastructure/eventstore/memory"
	"github.com/vardius/go-api-boilerplate/pkg/user/infrastructure/proto"
	user_proto "github.com/vardius/go-api-boilerplate/pkg/user/infrastructure/proto"
	server "github.com/vardius/go-api-boilerplate/pkg/user/interfaces/grpc"
	user_http "github.com/vardius/go-api-boilerplate/pkg/user/interfaces/http"
	"github.com/vardius/gorouter"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

type config struct {
	Env      string   `env:"ENV"       envDefault:"development"`
	Host     string   `env:"HOST"      envDefault:"0.0.0.0"`
	PortHTTP int      `env:"PORT_HTTP" envDefault:"3020"`
	PortGRPC int      `env:"PORT_GRPC" envDefault:"3021"`
	Secret   string   `env:"SECRET"    envDefault:"secret"`
	Origins  []string `env:"ORIGINS"   envSeparator:"|"` // Origins should follow format: scheme "://" host [ ":" port ]
}

func main() {
	ctx := context.Background()

	cfg := config{}
	env.Parse(&cfg)

	logger := log.New(cfg.Env)
	rec := recovery.WithLogger(recovery.New(), logger)
	jwtService := jwt.New([]byte(cfg.Secret), time.Hour*24)
	auth := authenticator.WithToken(jwtService.Decode)

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
	userServer := server.NewServer(
		commandbus.NewLoggable(runtime.NumCPU(), logger),
		eventbus.NewLoggable(runtime.NumCPU(), logger),
		eventstore.New(),
		jwt.New([]byte(cfg.Secret), time.Hour*24),
	)

	proto.RegisterUserServer(grpcServer, userServer)

	healthServer := health.NewServer()
	healthServer.SetServingStatus("user", healthpb.HealthCheckResponse_SERVING)
	healthpb.RegisterHealthServer(grpcServer, healthServer)

	lis, err := net.Listen("tcp", fmt.Sprintf("%s:%d", cfg.Host, cfg.PortGRPC))
	if err != nil {
		logger.Critical(ctx, "tcp failed to listen %s:%d\n%v\n", cfg.Host, cfg.PortGRPC, err)
	} else {
		logger.Info(ctx, "tcp running at %s:%d\n", cfg.Host, cfg.PortGRPC)
	}

	go func() {
		logger.Critical(ctx, "failed to serve: %v\n", grpcServer.Serve(lis))
	}()

	userConn, err := grpc.Dial(fmt.Sprintf("%s:%d", cfg.Host, cfg.PortGRPC), grpc.WithInsecure())
	if err != nil {
		logger.Critical(ctx, "grpc user conn dial error: %v\n", err)
		os.Exit(1)
	}
	defer userConn.Close()

	grpUserClient := user_proto.NewUserClient(userConn)

	// Global middleware
	router := gorouter.New(
		logger.LogRequest,
		cors.Default().Handler,
		response.WithXSS,
		response.WithHSTS,
		response.AsJSON,
		auth.FromHeader("USER"),
		auth.FromQuery("authToken"),
		rec.RecoverHandler,
	)

	user_http.AddHealthCheckRoutes(router, logger, userConn)
	user_http.AddUserRoutes(router, grpUserClient)

	srv := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.Host, cfg.PortHTTP),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
		Handler:      router,
	}

	go func() {
		logger.Critical(ctx, "%v\n", srv.ListenAndServe())
	}()

	logger.Info(ctx, "htpp running at %s:%d\n", cfg.Host, cfg.PortHTTP)

	shutdown.GracefulStop(func() {
		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		logger.Info(ctx, "shutting down...\n")

		grpcServer.GracefulStop()

		if err := srv.Shutdown(ctx); err != nil {
			logger.Critical(ctx, "shutdown error: %v\n", err)
		} else {
			logger.Info(ctx, "gracefully stopped\n")
		}
	})
}
