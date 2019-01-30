package main

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"net/http"
	"os"
	"runtime"
	"time"

	"github.com/caarlos0/env"
	_ "github.com/go-sql-driver/mysql"
	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_recovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	"github.com/rs/cors"
	user_proto "github.com/vardius/go-api-boilerplate/cmd/user/infrastructure/proto"
	server "github.com/vardius/go-api-boilerplate/cmd/user/interfaces/grpc"
	user_http "github.com/vardius/go-api-boilerplate/cmd/user/interfaces/http"
	commandbus "github.com/vardius/go-api-boilerplate/pkg/commandbus/memory"
	eventbus "github.com/vardius/go-api-boilerplate/pkg/eventbus/memory"
	eventstore "github.com/vardius/go-api-boilerplate/pkg/eventstore/memory"
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
	Env      string   `env:"ENV"       envDefault:"development"`
	Host     string   `env:"HOST"      envDefault:"0.0.0.0"`
	PortHTTP int      `env:"PORT_HTTP" envDefault:"3020"`
	PortGRPC int      `env:"PORT_GRPC" envDefault:"3021"`
	DbHost   string   `env:"DB_HOST"   envDefault:"0.0.0.0"`
	DbPort   int      `env:"DB_PORT"   envDefault:"3306"`
	DbUser   string   `env:"DB_USER"   envDefault:"root"`
	DbPass   string   `env:"DB_PASS"   envDefault:"password"`
	DbName   string   `env:"DB_NAME"   envDefault:"goapiboilerplate"`
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
	grpcServer := getGRPCServer(logger)

	db := getDBConnection(ctx, cfg.DbHost, cfg.DbPort, cfg.DbUser, cfg.DbPass, cfg.DbName, logger)
	defer db.Close()

	userServer := server.NewServer(
		commandbus.NewLoggable(runtime.NumCPU(), logger),
		eventbus.NewLoggable(runtime.NumCPU(), logger),
		eventstore.New(),
		db,
		jwt.New([]byte(cfg.Secret), time.Hour*24),
	)

	userConn := getGRPCConnection(ctx, cfg.Host, cfg.PortGRPC, logger)
	defer userConn.Close()

	grpUserClient := user_proto.NewUserServiceClient(userConn)

	healthServer := health.NewServer()
	healthServer.SetServingStatus("user", health_proto.HealthCheckResponse_SERVING)

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

	user_proto.RegisterUserServiceServer(grpcServer, userServer)
	health_proto.RegisterHealthServer(grpcServer, healthServer)

	user_http.AddHealthCheckRoutes(router, logger, userConn, db)
	user_http.AddUserRoutes(router, grpUserClient)

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

func getDBConnection(ctx context.Context, host string, port int, user, pass, database string, logger golog.Logger) (db *sql.DB) {
	db, err := sql.Open("mysql", fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true", user, pass, host, port, database))
	if err != nil {
		logger.Critical(ctx, "mysql conn error: %v\n", err)
		os.Exit(1)
	}

	db.SetConnMaxLifetime(time.Minute * 5)
	db.SetMaxIdleConns(0)
	db.SetMaxOpenConns(5)

	return db
}
