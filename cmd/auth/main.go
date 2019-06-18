package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"time"

	http_cors "github.com/rs/cors"
	auth_config "github.com/vardius/go-api-boilerplate/cmd/auth/application/config"
	auth_eventhandler "github.com/vardius/go-api-boilerplate/cmd/auth/application/eventhandler"
	auth_oauth2 "github.com/vardius/go-api-boilerplate/cmd/auth/application/oauth2"
	auth_client "github.com/vardius/go-api-boilerplate/cmd/auth/domain/client"
	auth_token "github.com/vardius/go-api-boilerplate/cmd/auth/domain/token"
	auth_persistence "github.com/vardius/go-api-boilerplate/cmd/auth/infrastructure/persistence/mysql"
	auth_proto "github.com/vardius/go-api-boilerplate/cmd/auth/infrastructure/proto"
	auth_repository "github.com/vardius/go-api-boilerplate/cmd/auth/infrastructure/repository"
	auth_grpc "github.com/vardius/go-api-boilerplate/cmd/auth/interfaces/grpc"
	auth_http "github.com/vardius/go-api-boilerplate/cmd/auth/interfaces/http"
	pubsub_proto "github.com/vardius/go-api-boilerplate/cmd/pubsub/infrastructure/proto"
	commandbus "github.com/vardius/go-api-boilerplate/pkg/commandbus"
	eventbus "github.com/vardius/go-api-boilerplate/pkg/eventbus"
	eventstore "github.com/vardius/go-api-boilerplate/pkg/eventstore/memory"
	grpc_utils "github.com/vardius/go-api-boilerplate/pkg/grpc"
	http_recovery "github.com/vardius/go-api-boilerplate/pkg/http/recovery"
	http_response "github.com/vardius/go-api-boilerplate/pkg/http/response"
	"github.com/vardius/go-api-boilerplate/pkg/log"
	"github.com/vardius/go-api-boilerplate/pkg/mysql"
	os_shutdown "github.com/vardius/go-api-boilerplate/pkg/os/shutdown"
	"github.com/vardius/gollback"
	"github.com/vardius/gorouter/v4"
	"google.golang.org/grpc"
	grpc_health "google.golang.org/grpc/health"
	grpc_health_proto "google.golang.org/grpc/health/grpc_health_v1"
	oauth2_models "gopkg.in/oauth2.v3/models"
)

func main() {
	ctx := context.Background()

	logger := log.New(auth_config.Env.Environment)

	db := mysql.NewConnection(ctx, auth_config.Env.DbHost, auth_config.Env.DbPort, auth_config.Env.DbUser, auth_config.Env.DbPass, auth_config.Env.DbName, logger)
	defer db.Close()

	pubsubConn := grpc_utils.NewConnection(ctx, auth_config.Env.PubSubHost, auth_config.Env.PortGRPC, logger)
	defer pubsubConn.Close()

	grpPubSubClient := pubsub_proto.NewMessageBusClient(pubsubConn)

	eventStore := eventstore.New()
	commandBus := commandbus.New(auth_config.Env.CommandBusQueueSize, logger)
	eventBus := eventbus.New(grpPubSubClient, logger)

	tokenRepository := auth_repository.NewTokenRepository(eventStore, eventBus)
	clientRepository := auth_repository.NewClientRepository(eventStore, eventBus)

	tokenMYSQLRepository := auth_persistence.NewTokenRepository(db)
	clientMYSQLRepository := auth_persistence.NewClientRepository(db)

	tokenStore := auth_oauth2.NewTokenStore(tokenMYSQLRepository, commandBus)
	clientStore := auth_oauth2.NewClientStore(clientMYSQLRepository)

	// store our internal user service client
	clientStore.SetInternal(auth_config.Env.UserClientID, &oauth2_models.Client{
		ID:     auth_config.Env.UserClientID,
		Secret: auth_config.Env.UserClientSecret,
		Domain: fmt.Sprintf("http://%s:%d", auth_config.Env.UserHost, auth_config.Env.PortHTTP),
	})

	manager := auth_oauth2.NewManager(tokenStore, clientStore, []byte(auth_config.Env.Secret))
	oauth2Server := auth_oauth2.InitServer(manager, db, logger, auth_config.Env.Secret)

	grpcServer := grpc_utils.NewServer(logger)
	authServer := auth_grpc.NewServer(oauth2Server, auth_config.Env.Secret)

	authConn := grpc_utils.NewConnection(ctx, auth_config.Env.Host, auth_config.Env.PortGRPC, logger)
	defer authConn.Close()

	healthServer := grpc_health.NewServer()
	healthServer.SetServingStatus("auth", grpc_health_proto.HealthCheckResponse_SERVING)

	http_recovery.WithLogger(logger)
	http_response.WithLogger(logger)

	// Global middleware
	router := gorouter.New(
		logger.LogRequest,
		http_cors.Default().Handler,
		http_response.WithXSS,
		http_response.WithHSTS,
		http_response.AsJSON,
		http_recovery.WithRecover,
	)

	auth_proto.RegisterAuthenticationServiceServer(grpcServer, authServer)
	grpc_health_proto.RegisterHealthServer(grpcServer, healthServer)

	auth_http.AddHealthCheckRoutes(router, db, map[string]*grpc.ClientConn{
		"auth":   authConn,
		"pubsub": pubsubConn,
	})
	auth_http.AddAuthRoutes(router, oauth2Server)

	srv := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", auth_config.Env.Host, auth_config.Env.PortHTTP),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
		Handler:      router,
	}

	lis, err := net.Listen("tcp", fmt.Sprintf("%s:%d", auth_config.Env.Host, auth_config.Env.PortGRPC))
	if err != nil {
		logger.Critical(ctx, "tcp failed to listen %s:%d\n%v\n", auth_config.Env.Host, auth_config.Env.PortGRPC, err)
		os.Exit(1)
	}

	commandBus.Subscribe((auth_token.Create{}).GetName(), auth_token.OnCreate(tokenRepository, db))
	commandBus.Subscribe((auth_token.Remove{}).GetName(), auth_token.OnRemove(tokenRepository, db))
	commandBus.Subscribe((auth_client.Create{}).GetName(), auth_client.OnCreate(clientRepository, db))
	commandBus.Subscribe((auth_client.Remove{}).GetName(), auth_client.OnRemove(clientRepository, db))

	go func() {
		gb := gollback.New(ctx)
		for {
			if grpc_utils.IsConnectionServing("pubsub", pubsubConn) {
				// Will resubscribe to handler on error infinitely
				go gb.Retry(0, func(ctx context.Context) (interface{}, error) {
					return nil, eventBus.Subscribe(ctx, (auth_token.WasCreated{}).GetType(), auth_eventhandler.WhenTokenWasCreated(db, tokenMYSQLRepository))
				})
				go gb.Retry(0, func(ctx context.Context) (interface{}, error) {
					return nil, eventBus.Subscribe(ctx, (auth_token.WasRemoved{}).GetType(), auth_eventhandler.WhenTokenWasRemoved(db, tokenMYSQLRepository))
				})
				go gb.Retry(0, func(ctx context.Context) (interface{}, error) {
					return nil, eventBus.Subscribe(ctx, (auth_client.WasCreated{}).GetType(), auth_eventhandler.WhenClientWasCreated(db, clientMYSQLRepository))
				})
				go gb.Retry(0, func(ctx context.Context) (interface{}, error) {
					return nil, eventBus.Subscribe(ctx, (auth_client.WasRemoved{}).GetType(), auth_eventhandler.WhenClientWasRemoved(db, clientMYSQLRepository))
				})
				break
			}
			time.Sleep(1 * time.Second)
		}
	}()

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

	logger.Info(ctx, "tcp running at %s:%d\n", auth_config.Env.Host, auth_config.Env.PortGRPC)
	logger.Info(ctx, "http running at %s:%d\n", auth_config.Env.Host, auth_config.Env.PortHTTP)

	os_shutdown.GracefulStop(stop)
}
