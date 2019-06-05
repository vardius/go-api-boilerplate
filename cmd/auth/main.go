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
	http_cors "github.com/rs/cors"
	"github.com/vardius/go-api-boilerplate/cmd/auth/application"
	auth_oauth2 "github.com/vardius/go-api-boilerplate/cmd/auth/application/oauth2"
	auth_client "github.com/vardius/go-api-boilerplate/cmd/auth/domain/client"
	auth_token "github.com/vardius/go-api-boilerplate/cmd/auth/domain/token"
	auth_persistence "github.com/vardius/go-api-boilerplate/cmd/auth/infrastructure/persistence/mysql"
	auth_proto "github.com/vardius/go-api-boilerplate/cmd/auth/infrastructure/proto"
	auth_repository "github.com/vardius/go-api-boilerplate/cmd/auth/infrastructure/repository"
	auth_grpc "github.com/vardius/go-api-boilerplate/cmd/auth/interfaces/grpc"
	auth_http "github.com/vardius/go-api-boilerplate/cmd/auth/interfaces/http"
	commandbus "github.com/vardius/go-api-boilerplate/pkg/commandbus/memory"
	eventbus "github.com/vardius/go-api-boilerplate/pkg/eventbus/memory"
	eventstore "github.com/vardius/go-api-boilerplate/pkg/eventstore/memory"
	"github.com/vardius/go-api-boilerplate/pkg/grpc"
	http_recovery "github.com/vardius/go-api-boilerplate/pkg/http/recovery"
	http_response "github.com/vardius/go-api-boilerplate/pkg/http/response"
	"github.com/vardius/go-api-boilerplate/pkg/log"
	"github.com/vardius/go-api-boilerplate/pkg/mysql"
	os_shutdown "github.com/vardius/go-api-boilerplate/pkg/os/shutdown"
	"github.com/vardius/gorouter/v4"
	grpc_health "google.golang.org/grpc/health"
	grpc_health_proto "google.golang.org/grpc/health/grpc_health_v1"
	oauth2_models "gopkg.in/oauth2.v3/models"
)

type config struct {
	Env              string   `env:"ENV"                 envDefault:"development"`
	Secret           string   `env:"SECRET"              envDefault:"secret"`
	Origins          []string `env:"ORIGINS"             envSeparator:"|"` // Origins should follow format: scheme "://" host [ ":" port ]
	Host             string   `env:"HOST"                envDefault:"0.0.0.0"`
	PortHTTP         int      `env:"PORT_HTTP"           envDefault:"3010"`
	PortGRPC         int      `env:"PORT_GRPC"           envDefault:"3011"`
	DbHost           string   `env:"DB_HOST"             envDefault:"0.0.0.0"`
	DbPort           int      `env:"DB_PORT"             envDefault:"3306"`
	DbUser           string   `env:"DB_USER"             envDefault:"root"`
	DbPass           string   `env:"DB_PASS"             envDefault:"password"`
	DbName           string   `env:"DB_NAME"             envDefault:"goapiboilerplate"`
	UserHost         string   `env:"USER_HOST"           envDefault:"0.0.0.0"`
	UserClientID     string   `env:"USER_CLIENT_ID"      envDefault:"clientId"`
	UserClientSecret string   `env:"USER_CLIENT_SECRET"  envDefault:"clientSecret"`
}

func main() {
	ctx := context.Background()

	cfg := config{}
	env.Parse(&cfg)

	logger := log.New(cfg.Env)

	db := mysql.NewConnection(ctx, cfg.DbHost, cfg.DbPort, cfg.DbUser, cfg.DbPass, cfg.DbName, logger)
	defer db.Close()

	eventStore := eventstore.New()
	commandBus := commandbus.NewLoggable(runtime.NumCPU(), logger)
	eventBus := eventbus.NewLoggable(runtime.NumCPU(), logger)

	tokenRepository := auth_repository.NewTokenRepository(eventStore, eventBus)
	clientRepository := auth_repository.NewClientRepository(eventStore, eventBus)

	tokenMYSQLRepository := auth_persistence.NewTokenRepository(db)
	clientMYSQLRepository := auth_persistence.NewClientRepository(db)

	commandBus.Subscribe(fmt.Sprintf("%T", &auth_token.Create{}), auth_token.OnCreate(tokenRepository, db))
	commandBus.Subscribe(fmt.Sprintf("%T", &auth_token.Remove{}), auth_token.OnRemove(tokenRepository, db))
	commandBus.Subscribe(fmt.Sprintf("%T", &auth_client.Create{}), auth_client.OnCreate(clientRepository, db))
	commandBus.Subscribe(fmt.Sprintf("%T", &auth_client.Remove{}), auth_client.OnRemove(clientRepository, db))

	eventBus.Subscribe(fmt.Sprintf("%T", &auth_token.WasCreated{}), application.WhenTokenWasCreated(db, tokenMYSQLRepository))
	eventBus.Subscribe(fmt.Sprintf("%T", &auth_token.WasRemoved{}), application.WhenTokenWasRemoved(db, tokenMYSQLRepository))
	eventBus.Subscribe(fmt.Sprintf("%T", &auth_client.WasCreated{}), application.WhenClientWasCreated(db, clientMYSQLRepository))
	eventBus.Subscribe(fmt.Sprintf("%T", &auth_client.WasRemoved{}), application.WhenClientWasRemoved(db, clientMYSQLRepository))

	tokenStore := auth_oauth2.NewTokenStore(tokenMYSQLRepository, commandBus)
	clientStore := auth_oauth2.NewClientStore(clientMYSQLRepository)

	// store our internal user service client
	clientStore.SetInternal(cfg.UserClientID, &oauth2_models.Client{
		ID:     cfg.UserClientID,
		Secret: cfg.UserClientSecret,
		Domain: fmt.Sprintf("http://%s:%d", cfg.UserHost, cfg.PortHTTP),
	})

	manager := auth_oauth2.NewManager(tokenStore, clientStore, []byte(cfg.Secret))
	oauth2Server := auth_oauth2.InitServer(manager, db, logger, cfg.Secret)

	grpcServer := grpc.NewServer(logger)
	authServer := auth_grpc.NewServer(oauth2Server, cfg.Secret)

	authConn := grpc.NewConnection(ctx, cfg.Host, cfg.PortGRPC, logger)
	defer authConn.Close()

	healthServer := grpc_health.NewServer()
	healthServer.SetServingStatus("auth", grpc_health_proto.HealthCheckResponse_SERVING)

	// Global middleware
	router := gorouter.New(
		logger.LogRequest,
		http_cors.Default().Handler,
		http_response.WithXSS,
		http_response.WithHSTS,
		http_response.AsJSON,
		http_recovery.WithLogger(logger).RecoverHandler,
	)

	auth_proto.RegisterAuthenticationServiceServer(grpcServer, authServer)
	grpc_health_proto.RegisterHealthServer(grpcServer, healthServer)

	auth_http.AddHealthCheckRoutes(router, logger, authConn)
	auth_http.AddAuthRoutes(router, oauth2Server)

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

	stop := func() {
		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		logger.Info(ctx, "shutting down...\n")

		grpcServer.GracefulStop()

		if err := srv.Shutdown(ctx); err != nil {
			logger.Critical(ctx, "shutdown error: %v\n", err)
		} else {
			logger.Info(ctx, "gracefully stopped\n")
		}
	}

	go func() {
		logger.Critical(ctx, "failed to serve: %v\n", grpcServer.Serve(lis))
		stop()
		os.Exit(1)
	}()

	go func() {
		logger.Critical(ctx, "%v\n", srv.ListenAndServe())
		stop()
		os.Exit(1)
	}()

	logger.Info(ctx, "tcp running at %s:%d\n", cfg.Host, cfg.PortGRPC)
	logger.Info(ctx, "http running at %s:%d\n", cfg.Host, cfg.PortHTTP)

	os_shutdown.GracefulStop(stop)
}
