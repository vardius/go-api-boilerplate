package main

import (
	"context"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql"
	config "github.com/vardius/go-api-boilerplate/cmd/auth/application/config"
	eventhandler "github.com/vardius/go-api-boilerplate/cmd/auth/application/eventhandler"
	oauth2 "github.com/vardius/go-api-boilerplate/cmd/auth/application/oauth2"
	router "github.com/vardius/go-api-boilerplate/cmd/auth/application/router"
	server "github.com/vardius/go-api-boilerplate/cmd/auth/application/server"
	client "github.com/vardius/go-api-boilerplate/cmd/auth/domain/client"
	token "github.com/vardius/go-api-boilerplate/cmd/auth/domain/token"
	persistence "github.com/vardius/go-api-boilerplate/cmd/auth/infrastructure/persistence/mysql"
	auth_proto "github.com/vardius/go-api-boilerplate/cmd/auth/infrastructure/proto"
	repository "github.com/vardius/go-api-boilerplate/cmd/auth/infrastructure/repository"
	auth_grpc "github.com/vardius/go-api-boilerplate/cmd/auth/interfaces/grpc"
	http "github.com/vardius/go-api-boilerplate/cmd/auth/interfaces/http"
	commandbus "github.com/vardius/go-api-boilerplate/pkg/commandbus"
	eventbus "github.com/vardius/go-api-boilerplate/pkg/eventbus"
	eventstore "github.com/vardius/go-api-boilerplate/pkg/eventstore/memory"
	grpc_utils "github.com/vardius/go-api-boilerplate/pkg/grpc"
	log "github.com/vardius/go-api-boilerplate/pkg/log"
	mysql "github.com/vardius/go-api-boilerplate/pkg/mysql"
	pubsub_proto "github.com/vardius/pubsub/proto"
	"google.golang.org/grpc"
	grpc_health "google.golang.org/grpc/health"
	grpc_health_proto "google.golang.org/grpc/health/grpc_health_v1"
	oauth2_models "gopkg.in/oauth2.v3/models"
)

func main() {
	ctx := context.Background()

	logger := log.New(config.Env.Environment)
	eventStore := eventstore.New()
	grpcServer := grpc_utils.NewServer(config.Env, logger)
	commandBus := commandbus.New(config.Env.CommandBusQueueSize, logger)

	mysqlConnection := mysql.NewConnection(ctx, config.Env, logger)
	defer mysqlConnection.Close()
	grpcPubsubConn := grpc_utils.NewConnection(ctx, config.Env.PubSubHost, config.Env.PortGRPC, config.Env, logger)
	defer grpcPubsubConn.Close()
	grpcAuthConn := grpc_utils.NewConnection(ctx, config.Env.Host, config.Env.PortGRPC, config.Env, logger)
	defer grpcAuthConn.Close()

	grpPubsubClient := pubsub_proto.NewMessageBusClient(grpcPubsubConn)
	eventBus := eventbus.New(grpPubsubClient, logger)
	tokenRepository := repository.NewTokenRepository(eventStore, eventBus)
	clientRepository := repository.NewClientRepository(eventStore, eventBus)
	tokenPersistenceRepository := persistence.NewTokenRepository(mysqlConnection)
	clientPersistenceRepository := persistence.NewClientRepository(mysqlConnection)
	tokenStore := oauth2.NewTokenStore(tokenPersistenceRepository, commandBus)
	clientStore := oauth2.NewClientStore(clientPersistenceRepository)
	manager := oauth2.NewManager(tokenStore, clientStore, []byte(config.Env.Secret))
	oauth2Server := oauth2.InitServer(manager, mysqlConnection, logger, config.Env.Secret)
	grpcHealthServer := grpc_health.NewServer()
	grpcAuthServer := auth_grpc.NewServer(oauth2Server, logger, config.Env.Secret)
	router := router.New(logger)
	server := server.New(
		logger,
		fmt.Sprintf("%s:%d", config.Env.Host, config.Env.PortHTTP),
		fmt.Sprintf("%s:%d", config.Env.Host, config.Env.PortGRPC),
	)

	// store our internal user service client
	clientStore.SetInternal(config.Env.UserClientID, &oauth2_models.Client{
		ID:     config.Env.UserClientID,
		Secret: config.Env.UserClientSecret,
		Domain: fmt.Sprintf("http://%s:%d", config.Env.UserHost, config.Env.PortHTTP),
	})

	commandBus.Subscribe((token.Create{}).GetName(), token.OnCreate(tokenRepository, mysqlConnection))
	commandBus.Subscribe((token.Remove{}).GetName(), token.OnRemove(tokenRepository, mysqlConnection))
	commandBus.Subscribe((client.Create{}).GetName(), client.OnCreate(clientRepository, mysqlConnection))
	commandBus.Subscribe((client.Remove{}).GetName(), client.OnRemove(clientRepository, mysqlConnection))

	go func() {
		go func() {
			eventhandler.Register(
				grpcPubsubConn,
				eventBus,
				map[string]eventbus.EventHandler{
					(token.WasCreated{}).GetType():  eventhandler.WhenTokenWasCreated(mysqlConnection, tokenPersistenceRepository),
					(token.WasRemoved{}).GetType():  eventhandler.WhenTokenWasRemoved(mysqlConnection, tokenPersistenceRepository),
					(client.WasCreated{}).GetType(): eventhandler.WhenClientWasCreated(mysqlConnection, clientPersistenceRepository),
					(client.WasRemoved{}).GetType(): eventhandler.WhenClientWasRemoved(mysqlConnection, clientPersistenceRepository),
				},
				5*time.Minute,
			)
		}()
	}()

	http.AddHealthCheckRoutes(
		router,
		mysqlConnection,
		map[string]*grpc.ClientConn{
			"auth":   grpcAuthConn,
			"pubsub": grpcPubsubConn,
		},
	)
	http.AddAuthRoutes(router, oauth2Server)

	auth_proto.RegisterAuthenticationServiceServer(grpcServer, grpcAuthServer)
	grpc_health_proto.RegisterHealthServer(grpcServer, grpcHealthServer)

	server.Run(ctx, router, grpcServer, grpcHealthServer)
}
