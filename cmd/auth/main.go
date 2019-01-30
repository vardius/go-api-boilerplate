package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/caarlos0/env"
	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_recovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	"github.com/rs/cors"
	"github.com/vardius/go-api-boilerplate/cmd/auth/infrastructure/proto"
	auth_proto "github.com/vardius/go-api-boilerplate/cmd/auth/infrastructure/proto"
	server "github.com/vardius/go-api-boilerplate/cmd/auth/interfaces/grpc"
	auth_http "github.com/vardius/go-api-boilerplate/cmd/auth/interfaces/http"
	user_proto "github.com/vardius/go-api-boilerplate/cmd/user/infrastructure/proto"
	"github.com/vardius/go-api-boilerplate/pkg/http/response"
	"github.com/vardius/go-api-boilerplate/pkg/jwt"
	"github.com/vardius/go-api-boilerplate/pkg/log"
	"github.com/vardius/go-api-boilerplate/pkg/os/shutdown"
	"github.com/vardius/go-api-boilerplate/pkg/recovery"
	"github.com/vardius/go-api-boilerplate/pkg/security/authenticator"
	"github.com/vardius/golog"
	"github.com/vardius/gorouter"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/health"
	health_proto "google.golang.org/grpc/health/grpc_health_v1"
)

type config struct {
	Env      string   `env:"ENV"            envDefault:"development"`
	Host     string   `env:"HOST"           envDefault:"0.0.0.0"`
	PortHTTP int      `env:"PORT_HTTP"      envDefault:"3010"`
	PortGRPC int      `env:"PORT_GRPC"      envDefault:"3011"`
	UserHost string   `env:"USER_HOST"      envDefault:"0.0.0.0"`
	Secret   string   `env:"SECRET"         envDefault:"secret"`
	Origins  []string `env:"ORIGINS"        envSeparator:"|"` // Origins should follow format: scheme "://" host [ ":" port ]
}

func main() {
	ctx := context.Background()

	cfg := config{}
	env.Parse(&cfg)

	logger := log.New(cfg.Env)
	rec := recovery.WithLogger(recovery.New(), logger)
	jwtService := jwt.New([]byte(cfg.Secret), time.Hour*24)
	auth := authenticator.WithToken(jwtService.Decode)
	grpcServer := getGRPCServer(logger)
	authServer := server.NewServer(jwtService)

	authConn := getGRPCConnection(ctx, cfg.Host, cfg.PortGRPC, logger)
	defer authConn.Close()

	userConn := getGRPCConnection(ctx, cfg.UserHost, cfg.PortGRPC, logger)
	defer userConn.Close()

	grpAuthClient := auth_proto.NewAuthenticationClient(authConn)
	grpUserClient := user_proto.NewUserServiceClient(userConn)

	healthServer := health.NewServer()
	healthServer.SetServingStatus("auth", health_proto.HealthCheckResponse_SERVING)

	// Global middleware
	router := gorouter.New(
		logger.LogRequest,
		cors.Default().Handler,
		response.WithXSS,
		response.WithHSTS,
		response.AsJSON,
		auth.FromHeader("AUTH"),
		auth.FromQuery("authToken"),
		rec.RecoverHandler,
	)

	proto.RegisterAuthenticationServer(grpcServer, authServer)
	health_proto.RegisterHealthServer(grpcServer, healthServer)

	auth_http.AddHealthCheckRoutes(router, logger, authConn, userConn)
	auth_http.AddAuthRoutes(router, grpUserClient, grpAuthClient)

	srv := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.Host, cfg.PortHTTP),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
		Handler:      router,
	}

	lis, err := net.Listen("tcp", fmt.Sprintf("%s:%d", cfg.Host, cfg.PortGRPC))
	if err != nil {
		logger.Critical(ctx, "tcp failed to listen %s:%d\n%v\n", cfg.Host, cfg.PortGRPC, err)
		os.Exit(1)
	}

	go func() {
		logger.Critical(ctx, "failed to serve: %v\n", grpcServer.Serve(lis))
	}()

	go func() {
		logger.Critical(ctx, "%v\n", srv.ListenAndServe())
	}()

	logger.Info(ctx, "tcp running at %s:%d\n", cfg.Host, cfg.PortGRPC)
	logger.Info(ctx, "http running at %s:%d\n", cfg.Host, cfg.PortHTTP)

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

func getGRPCServer(logger golog.Logger) *grpc.Server {
	opts := []grpc_recovery.Option{
		grpc_recovery.WithRecoveryHandlerContext(func(ctx context.Context, rec interface{}) (err error) {
			logger.Critical(ctx, "Recovered in f %v", rec)

			return grpc.Errorf(codes.Internal, "%s", rec)
		}),
	}

	server := grpc.NewServer(
		grpc_middleware.WithUnaryServerChain(
			grpc_recovery.UnaryServerInterceptor(opts...),
		),
		grpc_middleware.WithStreamServerChain(
			grpc_recovery.StreamServerInterceptor(opts...),
		),
	)

	return server
}

func getGRPCConnection(ctx context.Context, host string, port int, logger golog.Logger) *grpc.ClientConn {
	conn, err := grpc.DialContext(ctx, fmt.Sprintf("%s:%d", host, port), grpc.WithInsecure())
	if err != nil {
		logger.Critical(ctx, "grpc auth conn dial error: %v\n", err)
		os.Exit(1)
	}

	return conn
}
