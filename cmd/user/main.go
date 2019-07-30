package main

import (
	"context"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql"
	auth_proto "github.com/vardius/go-api-boilerplate/cmd/auth/infrastructure/proto"
	config "github.com/vardius/go-api-boilerplate/cmd/user/application/config"
	eventhandler "github.com/vardius/go-api-boilerplate/cmd/user/application/eventhandler"
	oauth2 "github.com/vardius/go-api-boilerplate/cmd/user/application/oauth2"
	router "github.com/vardius/go-api-boilerplate/cmd/user/application/router"
	user "github.com/vardius/go-api-boilerplate/cmd/user/domain/user"
	persistence "github.com/vardius/go-api-boilerplate/cmd/user/infrastructure/persistence/mysql"
	user_proto "github.com/vardius/go-api-boilerplate/cmd/user/infrastructure/proto"
	repository "github.com/vardius/go-api-boilerplate/cmd/user/infrastructure/repository"
	user_grpc "github.com/vardius/go-api-boilerplate/cmd/user/interfaces/grpc"
	http "github.com/vardius/go-api-boilerplate/cmd/user/interfaces/http"
	buildinfo "github.com/vardius/go-api-boilerplate/pkg/buildinfo"
	commandbus "github.com/vardius/go-api-boilerplate/pkg/commandbus"
	eventbus "github.com/vardius/go-api-boilerplate/pkg/eventbus"
	eventstore "github.com/vardius/go-api-boilerplate/pkg/eventstore/memory"
	grpc_utils "github.com/vardius/go-api-boilerplate/pkg/grpc"
	log "github.com/vardius/go-api-boilerplate/pkg/log"
	mysql "github.com/vardius/go-api-boilerplate/pkg/mysql"
	server "github.com/vardius/go-api-boilerplate/pkg/server"
	pubsub_proto "github.com/vardius/pubsub/proto"
	"google.golang.org/grpc"
	grpc_health "google.golang.org/grpc/health"
	grpc_health_proto "google.golang.org/grpc/health/grpc_health_v1"
)

func main() {
	buildinfo.PrintVersionOrContinue()

	ctx := context.Background()

	logger := log.New(config.Env.Environment)
	eventStore := eventstore.New()
	oauth2Config := oauth2.NewConfig()
	grpcServer := grpc_utils.NewServer(config.Env, logger)
	commandBus := commandbus.New(config.Env.CommandBusQueueSize, logger)

	mysqlConnection := mysql.NewConnection(ctx, config.Env, logger)
	defer mysqlConnection.Close()
	grpcPubsubConn := grpc_utils.NewConnection(ctx, config.Env.PubSubHost, config.Env.PortGRPC, config.Env, logger)
	defer grpcPubsubConn.Close()
	grpcAuthConn := grpc_utils.NewConnection(ctx, config.Env.AuthHost, config.Env.PortGRPC, config.Env, logger)
	defer grpcAuthConn.Close()
	grpcUserConn := grpc_utils.NewConnection(ctx, config.Env.Host, config.Env.PortGRPC, config.Env, logger)
	defer grpcUserConn.Close()

	grpcPubsubClient := pubsub_proto.NewMessageBusClient(grpcPubsubConn)
	eventBus := eventbus.New(grpcPubsubClient, logger)
	userPersistenceRepository := persistence.NewUserRepository(mysqlConnection)
	userRepository := repository.NewUserRepository(eventStore, eventBus)
	grpcAuthClient := auth_proto.NewAuthenticationServiceClient(grpcAuthConn)
	grpcHealthServer := grpc_health.NewServer()
	grpcUserServer := user_grpc.NewServer(commandBus, userPersistenceRepository, logger)
	router := router.New(logger, grpcAuthClient, userPersistenceRepository)
	server := server.New(
		logger,
		fmt.Sprintf("%s:%d", config.Env.Host, config.Env.PortHTTP),
		fmt.Sprintf("%s:%d", config.Env.Host, config.Env.PortGRPC),
	)

	commandBus.Subscribe((user.RegisterWithEmail{}).GetName(), user.OnRegisterWithEmail(userRepository, mysqlConnection))
	commandBus.Subscribe((user.RegisterWithGoogle{}).GetName(), user.OnRegisterWithGoogle(userRepository, mysqlConnection))
	commandBus.Subscribe((user.RegisterWithFacebook{}).GetName(), user.OnRegisterWithFacebook(userRepository, mysqlConnection))
	commandBus.Subscribe((user.ChangeEmailAddress{}).GetName(), user.OnChangeEmailAddress(userRepository, mysqlConnection))
	commandBus.Subscribe((user.RequestAccessToken{}).GetName(), user.OnRequestAccessToken(userRepository, mysqlConnection))

	go func() {
		eventhandler.Register(
			grpcPubsubConn,
			eventBus,
			map[string]eventbus.EventHandler{
				(user.WasRegisteredWithEmail{}).GetType():    eventhandler.WhenUserWasRegisteredWithEmail(mysqlConnection, userPersistenceRepository),
				(user.WasRegisteredWithGoogle{}).GetType():   eventhandler.WhenUserWasRegisteredWithGoogle(mysqlConnection, userPersistenceRepository),
				(user.WasRegisteredWithFacebook{}).GetType(): eventhandler.WhenUserWasRegisteredWithFacebook(mysqlConnection, userPersistenceRepository),
				(user.EmailAddressWasChanged{}).GetType():    eventhandler.WhenUserEmailAddressWasChanged(mysqlConnection, userPersistenceRepository),
				(user.AccessTokenWasRequested{}).GetType():   eventhandler.WhenUserAccessTokenWasRequested(oauth2Config, config.Env.Secret),
			},
			5*time.Minute,
		)
	}()

	http.AddHealthCheckRoutes(
		router,
		mysqlConnection,
		map[string]*grpc.ClientConn{
			"user":   grpcUserConn,
			"auth":   grpcAuthConn,
			"pubsub": grpcPubsubConn,
		},
	)
	http.AddAuthRoutes(router, commandBus, oauth2Config, config.Env.Secret)
	http.AddUserRoutes(router, commandBus, userPersistenceRepository)

	user_proto.RegisterUserServiceServer(grpcServer, grpcUserServer)
	grpc_health_proto.RegisterHealthServer(grpcServer, grpcHealthServer)

	server.Run(ctx, router, grpcServer, grpcHealthServer)
}
